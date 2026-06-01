import { describe, expect, it } from 'vitest';
import { getTranslatableEmailText, splitEmailTextForTranslation } from './emailTranslation';

describe('getTranslatableEmailText', () => {
  it('prefers plain text body over HTML body', () => {
    expect(
      getTranslatableEmailText({
        text_body: '  Plain body  ',
        html_body: '<p>HTML body</p>',
      })
    ).toBe('Plain body');
  });

  it('converts HTML body to readable plain text', () => {
    expect(
      getTranslatableEmailText({
        html_body:
          '<div>Hello <strong>world</strong></div><p>Next line</p><script>alert(1)</script>',
      })
    ).toBe('Hello world\nNext line');
  });

  it('does not translate snippets when no body exists', () => {
    expect(getTranslatableEmailText({ snippet: 'Only a preview' })).toBe('');
  });
});

describe('splitEmailTextForTranslation', () => {
  it('splits text into paragraph batches', () => {
    expect(
      splitEmailTextForTranslation('First paragraph.\n\nSecond paragraph.\n\nThird paragraph.')
    ).toEqual(['First paragraph.', 'Second paragraph.', 'Third paragraph.']);
  });

  it('removes common email tracking filler characters before batching', () => {
    expect(splitEmailTextForTranslation('Hello\u034f \u00ad\u034f \u200b\n\nWorld')).toEqual([
      'Hello',
      'World',
    ]);
  });

  it('splits an oversized paragraph into smaller batches', () => {
    expect(splitEmailTextForTranslation('abcdef', { maxBatchLength: 3 })).toEqual(['abc', 'def']);
  });
});
