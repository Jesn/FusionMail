import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from 'vitest';
import toast from 'react-hot-toast';
import { translationService } from '../../services/translationService';
import type { EmailDetail as EmailDetailType } from '../../types';
import { EmailDetail } from './EmailDetail';

vi.mock('react-hot-toast', () => ({
  default: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

vi.mock('../../services/translationService', () => ({
  translationService: {
    translateEmailText: vi.fn(),
  },
}));

const translateEmailTextMock = vi.mocked(translationService.translateEmailText);
const toastSuccessMock = toast.success as unknown as Mock;
const toastErrorMock = toast.error as unknown as Mock;

const baseEmail: EmailDetailType = {
  id: 1083,
  provider_id: 'provider-1',
  account_uid: 'account-1',
  message_id: 'message-1',
  from_address: 'sender@example.com',
  from_name: 'Sender',
  to_addresses: JSON.stringify(['recipient@example.com']),
  subject: 'Translation test',
  snippet: 'Preview text',
  is_read: true,
  is_starred: false,
  is_archived: false,
  is_deleted: false,
  is_spam: false,
  has_attachments: false,
  attachments_count: 0,
  sent_at: '2026-05-28T08:00:00Z',
  received_at: '2026-05-28T08:00:00Z',
  created_at: '2026-05-28T08:00:00Z',
  updated_at: '2026-05-28T08:00:00Z',
  text_body: 'Original body text',
};

function renderEmailDetail(email: EmailDetailType = baseEmail) {
  return render(
    <EmailDetail
      email={email}
      onToggleStar={vi.fn()}
      onArchive={vi.fn()}
      onDelete={vi.fn()}
      onBack={vi.fn()}
    />
  );
}

describe('EmailDetail translation', () => {
  beforeEach(() => {
    translateEmailTextMock.mockReset();
    toastSuccessMock.mockReset();
    toastErrorMock.mockReset();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('replaces the visible body with translated text and toggles back to original', async () => {
    const user = userEvent.setup();
    translateEmailTextMock.mockResolvedValue('翻译后的正文');

    renderEmailDetail();

    expect(screen.getByText('Original body text')).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /翻译/ }));

    await waitFor(() => {
      expect(screen.getByText('翻译后的正文')).toBeInTheDocument();
    });
    expect(screen.queryByText('Original body text')).not.toBeInTheDocument();
    expect(translateEmailTextMock).toHaveBeenCalledWith('Original body text');
    expect(toastSuccessMock).toHaveBeenCalledWith('翻译完成');

    await user.click(screen.getByRole('button', { name: /查看原文/ }));

    expect(screen.getByText('Original body text')).toBeInTheDocument();
    expect(screen.queryByText('翻译后的正文')).not.toBeInTheDocument();
  });

  it('translates the body paragraph by paragraph', async () => {
    const user = userEvent.setup();
    translateEmailTextMock
      .mockResolvedValueOnce('第一段译文')
      .mockResolvedValueOnce('第二段译文');

    renderEmailDetail({
      ...baseEmail,
      text_body: 'First paragraph.\n\nSecond paragraph.',
    });

    await user.click(screen.getByRole('button', { name: /翻译/ }));

    await waitFor(() => {
      expect(screen.getByText(/第一段译文/)).toBeInTheDocument();
    });
    expect(screen.getByText(/第二段译文/)).toBeInTheDocument();
    expect(translateEmailTextMock).toHaveBeenNthCalledWith(1, 'First paragraph.');
    expect(translateEmailTextMock).toHaveBeenNthCalledWith(2, 'Second paragraph.');
  });

  it('keeps the original body visible when translation fails', async () => {
    const user = userEvent.setup();
    vi.spyOn(console, 'error').mockImplementation(() => undefined);
    translateEmailTextMock.mockRejectedValue(new Error('provider failed'));

    renderEmailDetail();

    await user.click(screen.getByRole('button', { name: /翻译/ }));

    await waitFor(() => {
      expect(toastErrorMock).toHaveBeenCalledWith('翻译失败');
    });
    expect(screen.getByText('Original body text')).toBeInTheDocument();
    expect(screen.queryByText('翻译后的正文')).not.toBeInTheDocument();
  });
});
