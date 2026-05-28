import { api } from './api';

interface TranslateResponse {
  success: boolean;
  data?: {
    translated_text?: string;
  };
}

export const translationService = {
  translateEmailText: async (text: string): Promise<string> => {
    const response = await api.post<TranslateResponse>('/translate', {
      text,
      source_lang: 'auto',
      target_lang: 'ZH',
    });

    const translatedText = response.data?.translated_text?.trim();
    if (!translatedText) {
      throw new Error('Invalid translation response');
    }

    return translatedText;
  },
};
