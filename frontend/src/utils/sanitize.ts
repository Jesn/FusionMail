import DOMPurify from 'dompurify';

/**
 * DOMPurify 安全配置
 */
const SANITIZE_CONFIG = {
  // 允许的 HTML 标签
  ALLOWED_TAGS: [
    'a', 'abbr', 'b', 'blockquote', 'br', 'code', 'div', 'em', 'i', 'li', 'ol', 'p', 'pre',
    'span', 'strong', 'table', 'tbody', 'thead', 'tr', 'td', 'th', 'u', 'ul', 'img',
    'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'hr', 'sup', 'sub', 'strike', 's', 'dl', 'dt', 'dd'
  ],

  // 允许的属性
  ALLOWED_ATTR: [
    'href', 'src', 'alt', 'title', 'class', 'id', 'style', 'width', 'height',
    'align', 'colspan', 'rowspan', 'border', 'cellpadding', 'cellspacing', 'rel',
    'target', 'download'
  ],

  // 禁止的标签
  FORBID_TAGS: [
    'script', 'style', 'iframe', 'object', 'embed', 'link', 'meta',
    'base', 'form', 'input', 'button', 'select', 'textarea', 'video', 'audio',
    'source', 'track', 'canvas', 'svg', 'math', 'applet', 'frame', 'frameset',
    'noscript', 'noembed', 'plaintext'
  ],

  // 禁止的属性
  FORBID_ATTR: [
    'onload', 'onclick', 'onerror', 'onmouseover', 'onfocus', 'onblur',
    'onchange', 'onsubmit', 'onkeydown', 'onkeyup', 'onkeypress', 'onmouseenter', 'onmouseleave',
    'onabort', 'onafterprint', 'onbeforeunload', 'oncanplay', 'oncanplaythrough', 'oncontextmenu',
    'oncopy', 'oncut', 'ondblclick', 'ondrag', 'ondragend', 'ondragenter', 'ondragleave',
    'ondragover', 'ondragstart', 'ondrop', 'ondurationchange', 'onemptied', 'onended',
    'onerror', 'onhashchange', 'oninput', 'oninvalid', 'onmessage', 'onmousedown',
    'onmousemove', 'onmouseout', 'onmouseup', 'onmousewheel', 'onoffline', 'ononline',
    'onpageshow', 'onpaste', 'onpause', 'onplay', 'onplaying', 'onpopstate', 'onprogress',
    'onratechange', 'onreset', 'onresize', 'onscroll', 'onsearch', 'onseeked', 'onseeking',
    'onselect', 'onstalled', 'onstorage', 'onsubmit', 'onsuspend', 'ontimeupdate',
    'ontoggle', 'onunload', 'onvolumechange', 'onwaiting', 'onwheel'
  ],

  // 安全选项
  ALLOW_UNKNOWN_PROTOCOLS: false,
  SANITIZE_DOM: true,
  KEEP_CONTENT: true,
  RETURN_TRUSTED_TYPE: false,
  ALLOW_DATA_ATTR: false,
  ALLOW_ARIA_ATTR: true,
  ALLOW_UNKNOWN_ATTR: false,
};

/**
 * 标准 HTML 清理
 */
export const sanitizeHtml = (html: string): string => {
  if (!html || typeof html !== 'string') {
    return '';
  }

  try {
    return DOMPurify.sanitize(html, {
      ...SANITIZE_CONFIG,
    });
  } catch (error) {
    console.error('HTML 清理失败:', error);
    return '';
  }
};

/**
 * 严格模式 HTML 清理
 * 移除更多可能危险的内容
 */
export const sanitizeHtmlStrict = (html: string): string => {
  if (!html || typeof html !== 'string') {
    return '';
  }

  try {
    return DOMPurify.sanitize(html, {
      ...SANITIZE_CONFIG,
      ALLOWED_TAGS: [
        'p', 'br', 'strong', 'em', 'u', 'a', 'div', 'span', 'ul', 'ol', 'li',
        'table', 'tbody', 'thead', 'tr', 'td', 'th', 'code', 'pre', 'blockquote'
      ],
      ALLOWED_ATTR: ['href', 'class', 'style'],
      FORBID_ATTR: ['style'], // 严格模式下连 style 也移除
    });
  } catch (error) {
    console.error('HTML 严格清理失败:', error);
    return '';
  }
};

/**
 * 检查 HTML 是否包含危险内容
 */
export const isDangerousHtml = (html: string): boolean => {
  if (!html) return false;

  const dangerousPatterns = [
    /<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi,
    /<iframe\b[^<]*(?:(?!<\/iframe>)<[^<]*)*<\/iframe>/gi,
    /javascript:/gi,
    /on\w+\s*=/gi,
    /<object/gi,
    /<embed/gi,
    /<link/gi,
    /<style/gi,
    /<meta[^>]*http-equiv=["']refresh/gi,
  ];

  return dangerousPatterns.some(pattern => pattern.test(html));
};

/**
 * 移除 CSS 样式中的危险属性
 */
export const sanitizeStyles = (styleString: string): string => {
  if (!styleString) return '';

  try {
    const styles = styleString
      .split(';')
      .filter(s => {
        const prop = s.toLowerCase().trim();
        if (!prop) return false;

        // 移除可能危险的 CSS 属性
        const dangerousProps = [
          'behavior', 'binding', 'expression', 'filter', 'import',
          'position', 'float', 'clear', 'display', 'overflow',
          'margin', 'padding', 'border', 'width', 'height',
          'left', 'right', 'top', 'bottom', 'z-index'
        ];

        return !dangerousProps.some(dangerous => prop.startsWith(dangerous + ':'));
      })
      .join('; ');

    return styles;
  } catch (error) {
    console.error('CSS 清理失败:', error);
    return '';
  }
};
