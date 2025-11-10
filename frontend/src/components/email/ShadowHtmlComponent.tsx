import { useEffect, useRef, useState } from 'react';
import { sanitizeHtml } from '../../utils/sanitize';

// 统一处理链接：全部新窗口打开，并补齐安全属性
function patchLinks(root: ParentNode) {
  const links = root.querySelectorAll<HTMLAnchorElement>('a[href]');
  links.forEach((a) => {
    const href = a.getAttribute('href') || '';
    if (!href || href.startsWith('#')) return;
    // 允许的协议：http/https/mailto/tel，其余忽略（已由 sanitize 处理，这里再次兜底）
    if (!/^(https?:|mailto:|tel:)/i.test(href)) return;

    a.setAttribute('target', '_blank');
    const rel = a.getAttribute('rel') || '';
    const want = ['noopener', 'noreferrer', 'nofollow', 'ugc'];
    const merged = Array.from(new Set([...rel.split(/\s+/).filter(Boolean), ...want])).join(' ');
    a.setAttribute('rel', merged);
  });
}


interface ShadowHtmlComponentProps {
  htmlContent: string;
  className?: string;
  useStrictMode?: boolean;
}

/**
 * ShadowHtmlComponent - 使用 Shadow DOM 渲染 HTML 内容
 * 关键修复：
 * - 只在挂载时 attachShadow，避免重复 attach 导致 NotSupportedError
 * - 更新内容时仅替换 wrapper.innerHTML，不销毁 shadowRoot
 */
export const ShadowHtmlComponent = ({
  htmlContent,
  className = 'email-content',
  useStrictMode = false,
}: ShadowHtmlComponentProps) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const shadowRootRef = useRef<ShadowRoot | null>(null);
  const contentWrapperRef = useRef<HTMLDivElement | null>(null);
  const [useFallback, setUseFallback] = useState(false);

  // 仅在组件挂载时创建 ShadowRoot 和静态样式
  useEffect(() => {
    const host = containerRef.current;
    if (!host) return;

    if (!host.attachShadow) {
      console.warn('此浏览器不支持 Shadow DOM，使用降级方案');
      setUseFallback(true);
      return;
    }

    try {
      // 只在第一次挂载时创建 shadowRoot
      const shadowRoot = host.shadowRoot || host.attachShadow({ mode: 'open' });
      shadowRootRef.current = shadowRoot;

      // 插入样式（只插入一次）
      const style = document.createElement('style');
      style.textContent = `
        :host { display: block; width: 100%; }
        .shadow-content-wrapper { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: inherit; font-size: inherit; }
        .shadow-content-wrapper * { max-width: 100% !important; box-sizing: border-box !important; overflow-wrap: break-word !important; word-wrap: break-word !important; -webkit-word-break: break-word !important; }
        .shadow-content-wrapper img { max-width: 100% !important; height: auto !important; display: block; margin: 0 auto; }
        .shadow-content-wrapper table { table-layout: auto !important; width: 100% !important; max-width: 100% !important; border-collapse: collapse; }
        .shadow-content-wrapper td, .shadow-content-wrapper th { word-break: break-word !important; overflow-wrap: break-word !important; max-width: 100% !important; }
        .shadow-content-wrapper a { color: #0066cc; text-decoration: underline; }
        .shadow-content-wrapper a:hover { color: #0052a3; }
        .shadow-content-wrapper p { margin: 0 0 1em 0; }
        .shadow-content-wrapper ul, .shadow-content-wrapper ol { margin: 0 0 1em 0; padding-left: 2em; }
        .shadow-content-wrapper blockquote { margin: 0 0 1em 0; padding-left: 1em; border-left: 3px solid #ddd; color: #666; }
        .shadow-content-wrapper pre { background: #f5f5f5; padding: 1em; overflow-x: auto; border-radius: 4px; }
        .shadow-content-wrapper code { background: #f5f5f5; padding: 0.2em 0.4em; border-radius: 3px; font-family: 'Courier New', Courier, monospace; }
        .shadow-content-wrapper pre code { background: none; padding: 0; }
        .shadow-content-wrapper h1, .shadow-content-wrapper h2, .shadow-content-wrapper h3, .shadow-content-wrapper h4, .shadow-content-wrapper h5, .shadow-content-wrapper h6 { margin: 1.5em 0 0.5em 0; font-weight: 600; }
        .shadow-content-wrapper h1 { font-size: 1.5em; }
        .shadow-content-wrapper h2 { font-size: 1.3em; }
        .shadow-content-wrapper h3 { font-size: 1.2em; }
        .shadow-content-wrapper h4 { font-size: 1.1em; }
        .shadow-content-wrapper h5 { font-size: 1em; }
        .shadow-content-wrapper h6 { font-size: 0.9em; }
      `;
      shadowRoot.appendChild(style);

      // 创建内容容器（只创建一次）
      const wrapper = document.createElement('div');
      wrapper.className = 'shadow-content-wrapper';
      shadowRoot.appendChild(wrapper);
      contentWrapperRef.current = wrapper;
    } catch (error) {
      console.error('Shadow DOM 渲染失败:', error);
      setUseFallback(true);
    }

    // 卸载时清理
    return () => {
      if (shadowRootRef.current) {
        shadowRootRef.current.innerHTML = '';
        shadowRootRef.current = null;
        contentWrapperRef.current = null;
      }
    };
  }, []);

  // htmlContent 或模式变化时，仅更新内容
  useEffect(() => {
    if (useFallback) return;
    const wrapper = contentWrapperRef.current;
    if (!wrapper) return;

    try {
      const cleanHtml = sanitizeHtml(htmlContent);
      wrapper.innerHTML = cleanHtml;
      patchLinks(wrapper);
    } catch (error) {
      console.error('Shadow DOM 内容更新失败:', error);
      setUseFallback(true);
    }
  }, [htmlContent, useStrictMode, useFallback]);

  // Fallback 模式下：也为所有链接添加新窗口打开与安全属性
  useEffect(() => {
    if (!useFallback) return;
    const host = containerRef.current;
    if (!host) return;
    try {
      patchLinks(host);
    } catch {}
  }, [useFallback, htmlContent]);


  // 降级方案：如果不支持 Shadow DOM 或渲染失败，使用普通 div
  if (useFallback) {
    return (
      <div
        ref={containerRef}
        className={className}
        dangerouslySetInnerHTML={{ __html: sanitizeHtml(htmlContent) }}
      />
    );
  }

  return <div ref={containerRef} className={className} />;
};

export default ShadowHtmlComponent;
