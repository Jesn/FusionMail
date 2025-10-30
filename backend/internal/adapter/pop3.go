package adapter

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/mail"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/knadh/go-pop3"
	"golang.org/x/net/proxy"
)

type POP3Adapter struct {
	config *Config
	client *pop3.Client
	mu     sync.Mutex
}

func NewPOP3Adapter(config *Config) (*POP3Adapter, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if config.Credentials == nil {
		return nil, fmt.Errorf("credentials is required")
	}
	if config.Credentials.Host == "" {
		return nil, fmt.Errorf("POP3 host is required")
	}
	if config.Credentials.Email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if config.Credentials.Password == "" {
		return nil, fmt.Errorf("password is required")
	}
	if config.Credentials.Port == 0 {
		config.Credentials.Port = 995
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	return &POP3Adapter{config: config}, nil
}

func (a *POP3Adapter) Connect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.client != nil {
		a.client = nil
	}
	opt := pop3.Opt{
		Host:       a.config.Credentials.Host,
		Port:       a.config.Credentials.Port,
		TLSEnabled: a.config.Credentials.TLS,
	}
	client := pop3.New(opt)
	conn, err := client.NewConn()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	if err := conn.Auth(a.config.Credentials.Email, a.config.Credentials.Password); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	a.client = client
	return nil
}

func (a *POP3Adapter) dialWithProxy(ctx context.Context, addr string) (net.Conn, error) {
	proxyAddr := fmt.Sprintf("%s:%d", a.config.Proxy.Host, a.config.Proxy.Port)
	var dialer proxy.Dialer
	var err error
	switch strings.ToLower(a.config.Proxy.Type) {
	case "socks5":
		auth := &proxy.Auth{}
		if a.config.Proxy.Username != "" {
			auth.User = a.config.Proxy.Username
			auth.Password = a.config.Proxy.Password
		}
		dialer, err = proxy.SOCKS5("tcp", proxyAddr, auth, proxy.Direct)
	default:
		return nil, fmt.Errorf("unsupported proxy type: %s", a.config.Proxy.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy dialer: %w", err)
	}
	conn, err := dialer.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial through proxy: %w", err)
	}
	if a.config.Credentials.TLS {
		tlsConfig := &tls.Config{ServerName: a.config.Credentials.Host}
		tlsConn := tls.Client(conn, tlsConfig)
		if err := tlsConn.Handshake(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("TLS handshake failed: %w", err)
		}
		return tlsConn, nil
	}
	return conn, nil
}

func (a *POP3Adapter) Disconnect() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.client != nil {
		a.client = nil
	}
	return nil
}

func (a *POP3Adapter) FetchEmails(ctx context.Context, since time.Time, limit int) ([]*Email, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.client == nil {
		return nil, fmt.Errorf("not connected")
	}
	conn, err := a.client.NewConn()
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}
	defer conn.Quit()
	if err := conn.Auth(a.config.Credentials.Email, a.config.Credentials.Password); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	count, _, err := conn.Stat()
	if err != nil {
		return nil, fmt.Errorf("STAT failed: %w", err)
	}
	if count == 0 {
		return []*Email{}, nil
	}
	fetchCount := count
	if limit > 0 && limit < count {
		fetchCount = limit
	}
	emails := make([]*Email, 0, fetchCount)
	for i := count; i > count-fetchCount; i-- {
		select {
		case <-ctx.Done():
			return emails, ctx.Err()
		default:
		}
		email, err := a.fetchEmailByNumber(conn, i)
		if err != nil {
			continue
		}
		if !since.IsZero() && email.SentAt.Before(since) {
			continue
		}
		emails = append(emails, email)
	}
	return emails, nil
}

func (a *POP3Adapter) fetchEmailByNumber(conn *pop3.Conn, msgNum int) (*Email, error) {
	msgBuffer, err := conn.RetrRaw(msgNum)
	if err != nil {
		return nil, fmt.Errorf("RETR failed: %w", err)
	}
	msg, err := mail.ReadMessage(strings.NewReader(msgBuffer.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to parse message: %w", err)
	}
	email := &Email{
		ProviderID: strconv.Itoa(msgNum),
		MessageID:  msg.Header.Get("Message-ID"),
		Subject:    msg.Header.Get("Subject"),
	}
	if from, err := mail.ParseAddress(msg.Header.Get("From")); err == nil {
		email.FromAddress = from.Address
		email.FromName = from.Name
	} else {
		email.FromAddress = msg.Header.Get("From")
	}
	if toAddrs, err := mail.ParseAddressList(msg.Header.Get("To")); err == nil {
		email.ToAddresses = make([]string, len(toAddrs))
		for i, addr := range toAddrs {
			email.ToAddresses[i] = addr.Address
		}
	}
	if ccAddrs, err := mail.ParseAddressList(msg.Header.Get("Cc")); err == nil {
		email.CcAddresses = make([]string, len(ccAddrs))
		for i, addr := range ccAddrs {
			email.CcAddresses[i] = addr.Address
		}
	}
	if date, err := msg.Header.Date(); err == nil {
		email.SentAt = date
		email.ReceivedAt = date
	} else {
		email.SentAt = time.Now()
		email.ReceivedAt = time.Now()
	}
	body, err := io.ReadAll(msg.Body)
	if err == nil {
		email.TextBody = string(body)
		if len(email.TextBody) > 200 {
			email.Snippet = email.TextBody[:200] + "..."
		} else {
			email.Snippet = email.TextBody
		}
	}
	email.InReplyTo = msg.Header.Get("In-Reply-To")
	email.References = msg.Header.Get("References")
	email.ReplyTo = msg.Header.Get("Reply-To")
	return email, nil
}

func (a *POP3Adapter) FetchEmailDetail(ctx context.Context, providerID string) (*Email, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.client == nil {
		return nil, fmt.Errorf("not connected")
	}
	conn, err := a.client.NewConn()
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}
	defer conn.Quit()
	if err := conn.Auth(a.config.Credentials.Email, a.config.Credentials.Password); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	msgNum, err := strconv.Atoi(providerID)
	if err != nil {
		return nil, fmt.Errorf("invalid provider ID: %s", providerID)
	}
	return a.fetchEmailByNumber(conn, msgNum)
}

func (a *POP3Adapter) GetProviderType() string {
	return a.config.Provider
}

func (a *POP3Adapter) GetProtocol() string {
	return "pop3"
}

func (a *POP3Adapter) TestConnection(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.client == nil {
		return fmt.Errorf("not connected")
	}
	conn, err := a.client.NewConn()
	if err != nil {
		return fmt.Errorf("failed to create connection: %w", err)
	}
	defer conn.Quit()
	if err := conn.Auth(a.config.Credentials.Email, a.config.Credentials.Password); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	_, _, err = conn.Stat()
	if err != nil {
		return fmt.Errorf("STAT failed: %w", err)
	}
	return nil
}
