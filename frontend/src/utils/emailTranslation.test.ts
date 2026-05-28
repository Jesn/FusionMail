import { describe, expect, it } from 'vitest';
import { getTranslatableEmailText } from './emailTranslation';

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
