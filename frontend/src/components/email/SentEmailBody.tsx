import type { SentEmail } from '../../types';
import ShadowHtmlComponent from './ShadowHtmlComponent';

type SentEmailBodyProps = {
  email: Pick<SentEmail, 'html_body' | 'text_body'>;
};

/**
 * 已发送邮件正文渲染入口。
 * HTML 正文必须经过 ShadowHtmlComponent 的清理和样式隔离，避免直接注入 DOM。
 */
export const SentEmailBody = ({ email }: SentEmailBodyProps) => {
  if (email.html_body) {
    return (
      <ShadowHtmlComponent
        className="prose prose-sm max-w-none dark:prose-invert"
        htmlContent={email.html_body}
      />
    );
  }

  if (email.text_body) {
    return (
      <pre className="whitespace-pre-wrap text-sm font-sans">
        {email.text_body}
      </pre>
    );
  }

  return <p className="text-muted-foreground text-sm">(无内容)</p>;
};

export default SentEmailBody;
