import { beforeEach, describe, expect, it, vi } from 'vitest';
import { api } from './api';
import { translationService } from './translationService';

vi.mock('./api', () => ({
  api: {
    post: vi.fn(),
  },
}));

const postMock = vi.mocked(api.post);

describe('translationService', () => {
  beforeEach(() => {
    postMock.mockReset();
  });

  it('posts text to the backend translation proxy', async () => {
    postMock.mockResolvedValue({
      success: true,
      data: { translated_text: '你好' },
    });

    const result = await translationService.translateEmailText('Hello');

    expect(result).toBe('你好');
    expect(postMock).toHaveBeenCalledWith('/translate', {
      text: 'Hello',
      source_lang: 'auto',
      target_lang: 'ZH',
    });
  });

  it('rejects malformed translation responses', async () => {
    postMock.mockResolvedValue({ success: true, data: {} });

    await expect(translationService.translateEmailText('Hello')).rejects.toThrow(
      'Invalid translation response'
    );
  });
});
