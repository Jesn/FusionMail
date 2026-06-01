type TranslatableEmail = {
  text_body?: string;
  html_body?: string;
  snippet?: string;
};

interface SplitEmailTextOptions {
  maxBatchLength?: number;
}

const defaultMaxBatchLength = 4000;

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
    .replace(/[\u00ad\u034f\u061c\u180e\u200b-\u200f\u202a-\u202e\u2060-\u206f\ufeff]/g, '')
    .replace(/\u00a0/g, ' ')
    .replace(/\r\n/g, '\n')
    .replace(/[ \t]+\n/g, '\n')
    .replace(/\n[ \t]+/g, '\n')
    .replace(/[ \t]{2,}/g, ' ')
    .replace(/\n{3,}/g, '\n\n')
    .trim();
}

function splitLongParagraph(paragraph: string, maxBatchLength: number): string[] {
  if (paragraph.length <= maxBatchLength) return [paragraph];

  const chunks: string[] = [];
  for (let index = 0; index < paragraph.length; index += maxBatchLength) {
    const chunk = paragraph.slice(index, index + maxBatchLength).trim();
    if (chunk) chunks.push(chunk);
  }
  return chunks;
}

export function splitEmailTextForTranslation(
  text: string,
  options: SplitEmailTextOptions = {}
): string[] {
  const maxBatchLength = Math.max(1, options.maxBatchLength ?? defaultMaxBatchLength);
  const normalized = normalizeEmailText(text);
  if (!normalized) return [];

  return normalized
    .split(/\n{2,}/)
    .flatMap((paragraph) => splitLongParagraph(normalizeEmailText(paragraph), maxBatchLength))
    .filter(Boolean);
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
