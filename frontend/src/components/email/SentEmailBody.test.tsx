import { render, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { SentEmailBody } from './SentEmailBody';

describe('SentEmailBody', () => {
  it('渲染 HTML 正文前会清理脚本和事件属性', async () => {
    const { container } = render(
      <SentEmailBody
        email={{
          html_body:
            '<p>安全内容</p><img src="x" onerror="alert(1)"><script>alert(2)</script>',
        }}
      />
    );

    const host = container.querySelector('.prose') as HTMLDivElement | null;
    expect(host).not.toBeNull();

    await waitFor(() => {
      expect(host?.shadowRoot?.textContent).toContain('安全内容');
    });

    const renderedHtml = host?.shadowRoot?.innerHTML ?? '';
    expect(renderedHtml).not.toContain('<script');
    expect(renderedHtml).not.toContain('onerror');
    expect(renderedHtml).toContain('安全内容');
  });

  it('纯文本正文使用文本节点渲染', () => {
    const { getByText } = render(
      <SentEmailBody email={{ text_body: '<script>alert(1)</script>' }} />
    );

    expect(getByText('<script>alert(1)</script>')).toBeInTheDocument();
  });
});
