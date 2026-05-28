type TranslatableEmail = {
  text_body?: string;
  html_body?: string;
  snippet?: string;
};

const blockSelector = [
  'address',
  'article',
  'aside',
  'blockquote',
  'br',
  'div',
  'footer',
  'h1',
  'h2',
  'h3',
  'h4',
  'h5',
  'h6',
  'header',
  'li',
  'main',
  'p',
  'section',
  'tr',
].join(',');

export function normalizeEmailText(text: string): string {
  return text
    .replace(/\u00a0/g, ' ')
    .replace(/\r\n/g, '\n')
    .replace(/[ \t]+\n/g, '\n')
    .replace(/\n[ \t]+/g, '\n')
    .replace(/[ \t]{2,}/g, ' ')
    .replace(/\n{3,}/g, '\n\n')
    .trim();
}

export function htmlToPlainText(html: string): string {
  const rawHtml = html.trim();
  if (!rawHtml) return '';

  if (typeof document === 'undefined') {
    return normalizeEmailText(
      rawHtml
        .replace(/<\s*(br|\/p|\/div|\/li|\/tr|\/h[1-6])\b[^>]*>/gi, '\n')
        .replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
        .replace(/<style\b[^<]*(?:(?!<\/style>)<[^<]*)*<\/style>/gi, '')
        .replace(/<[^>]+>/g, ' ')
    );
  }

  const template = document.createElement('template');
  template.innerHTML = rawHtml;
  template.content.querySelectorAll('script, style, noscript').forEach((node) => node.remove());
  template.content.querySelectorAll(blockSelector).forEach((node) => {
    node.appendChild(document.createTextNode('\n'));
  });

  return normalizeEmailText(template.content.textContent || '');
}

export function getTranslatableEmailText(email: TranslatableEmail): string {
  const textBody = normalizeEmailText(email.text_body || '');
  if (textBody) return textBody;

  if (email.html_body?.trim()) {
    return htmlToPlainText(email.html_body);
  }

  return '';
}
