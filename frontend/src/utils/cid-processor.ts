import type { EmailAttachment } from '../types';

interface ProcessedAttachment {
  id: number;
  contentId: string;
  blobUrl: string;
  filename: string;
  contentType: string;
}

/**
 * CID 资源处理器
 * 用于处理邮件中的内嵌图片（Content-ID）
 * CID 格式示例：image001@domain.com
 */
export class CidProcessor {
  private cidMap: Map<string, string> = new Map();
  private processedAttachments: ProcessedAttachment[] = [];
  private blobUrls: Set<string> = new Set();

  /**
   * 从附件中提取 CID 映射
   * @param attachments 邮件附件列表
   * @returns CID 到 Blob URL 的映射
   */
  extractCidMap(attachments: EmailAttachment[]): Map<string, string> {
    // 清空之前的数据
    this.cleanup();

    if (!attachments || !Array.isArray(attachments)) {
      return this.cidMap;
    }

    attachments.forEach(attachment => {
      // 提取 Content-ID
      const contentId = this.extractContentId(attachment);

      if (contentId && this.isImageAttachment(attachment)) {
        try {
          // 创建 Blob URL
          const blobUrl = this.createBlobUrl(attachment);

          if (blobUrl) {
            this.cidMap.set(contentId, blobUrl);
            this.blobUrls.add(blobUrl);

            // 保存处理过的附件信息
            this.processedAttachments.push({
              id: attachment.id,
              contentId,
              blobUrl,
              filename: attachment.filename,
              contentType: attachment.content_type,
            });
          }
        } catch (error) {
          console.error(`创建 CID 资源失败 (${attachment.filename}):`, error);
        }
      }
    });

    return this.cidMap;
  }

  /**
   * 替换 HTML 中的 CID 引用
   * 将 cid:contentId 替换为实际的 blob URL
   * @param html HTML 字符串
   * @param cidMap 可选，自定义 CID 映射
   * @returns 替换后的 HTML
   */
  replaceCidInHtml(html: string, cidMap?: Map<string, string>): string {
    if (!html || typeof html !== 'string') {
      return html;
    }

    const map = cidMap || this.cidMap;

    if (!map || map.size === 0) {
      return html;
    }

    let processedHtml = html;

    // 遍历所有 CID 映射，替换 HTML 中的引用
    map.forEach((blobUrl, cid) => {
      // 转义正则特殊字符
      const escapedCid = this.escapeRegExp(cid);

      // 匹配格式：src="cid:xxx" 或 src='cid:xxx' 或 src=cid:xxx
      const cidPattern = new RegExp(
        `(?:src|href)=["']?cid:${escapedCid}["']?`,
        'gi'
      );

      processedHtml = processedHtml.replace(cidPattern, `${blobUrl}`);
    });

    return processedHtml;
  }

  /**
   * 获取处理过的附件列表
   * @returns 处理过的附件数组
   */
  getProcessedAttachments(): ProcessedAttachment[] {
    return [...this.processedAttachments];
  }

  /**
   * 获取 Blob URL 数量
   * @returns 当前管理的 Blob URL 数量
   */
  getBlobUrlCount(): number {
    return this.blobUrls.size;
  }

  /**
   * 清理所有资源
   * 重要：调用此方法释放所有 Blob URL，避免内存泄漏
   */
  cleanup(): void {
    // 撤销所有 Blob URL
    this.blobUrls.forEach(blobUrl => {
      try {
        URL.revokeObjectURL(blobUrl);
      } catch (error) {
        console.warn('撤销 Blob URL 失败:', error);
      }
    });

    // 清空所有映射
    this.blobUrls.clear();
    this.cidMap.clear();
    this.processedAttachments = [];
  }

  /**
   * 从附件中提取 Content-ID
   * @param attachment 邮件附件
   * @returns Content-ID 字符串或 null
   */
  private extractContentId(attachment: EmailAttachment): string | null {
    // 尝试从多个字段提取 content_id
    if (attachment.content_id) {
      return attachment.content_id;
    }

    // 如果没有直接的 content_id，尝试从其他字段解析
    // 注意：headers 是 Record<string, string> 类型，不是字符串
    // 所以我们需要检查特定的键值对
    if (attachment.headers && typeof attachment.headers['content-id'] === 'string') {
      const contentId = attachment.headers['content-id'];
      // 提取尖括号内的值
      const match = contentId.match(/<([^>]+)>/);
      if (match && match[1]) {
        return match[1];
      }
    }

    return null;
  }

  /**
   * 检查附件是否为图片类型
   * @param attachment 邮件附件
   * @returns 是否为图片附件
   */
  private isImageAttachment(attachment: EmailAttachment): boolean {
    if (!attachment.content_type) {
      return false;
    }

    const imageTypes = [
      'image/jpeg',
      'image/jpg',
      'image/png',
      'image/gif',
      'image/webp',
      'image/svg+xml',
      'image/bmp',
      'image/tiff',
      'image/x-icon',
    ];

    const contentType = attachment.content_type.toLowerCase();
    return imageTypes.some(type => contentType.includes(type));
  }

  /**
   * 为附件创建 Blob URL
   * @param attachment 邮件附件
   * @returns Blob URL 字符串
   */
  private createBlobUrl(attachment: EmailAttachment): string | null {
    try {
      // 如果已经有 data 字段，直接使用
      if (attachment.data) {
        const blob = this.base64ToBlob(attachment.data, attachment.content_type);
        return URL.createObjectURL(blob);
      }

      // 如果有 content 字段，尝试使用
      if ((attachment as any).content) {
        const content = (attachment as any).content;
        if (content.startsWith('data:')) {
          // 如果是 data URL，转换为 blob
          const blob = this.dataUrlToBlob(content);
          return URL.createObjectURL(blob);
        } else if (content.startsWith('base64,')) {
          // 如果是 base64 数据
          const base64Data = content.replace('base64,', '');
          const blob = this.base64ToBlob(base64Data, attachment.content_type);
          return URL.createObjectURL(blob);
        }
      }

      // 如果都没有，可能需要通过 API 下载
      console.warn(`附件 ${attachment.filename} 无法直接创建 Blob URL，需要通过 API 下载`);
      return null;

    } catch (error) {
      console.error(`创建 Blob URL 失败 (${attachment.filename}):`, error);
      return null;
    }
  }

  /**
   * 将 Base64 字符串转换为 Blob
   * @param base64String Base64 字符串
   * @param contentType 内容类型
   * @returns Blob 对象
   */
  private base64ToBlob(base64String: string, contentType: string): Blob {
    const byteCharacters = atob(base64String);
    const byteNumbers = new Array(byteCharacters.length);

    for (let i = 0; i < byteCharacters.length; i++) {
      byteNumbers[i] = byteCharacters.charCodeAt(i);
    }

    const byteArray = new Uint8Array(byteNumbers);
    return new Blob([byteArray], { type: contentType });
  }

  /**
   * 将 Data URL 转换为 Blob
   * @param dataUrl Data URL
   * @returns Blob 对象
   */
  private dataUrlToBlob(dataUrl: string): Blob {
    const parts = dataUrl.split(',');
    const mimeMatch = parts[0].match(/:(.*?);/);
    const mime = mimeMatch ? mimeMatch[1] : 'application/octet-stream';

    const base64Data = parts[1];
    return this.base64ToBlob(base64Data, mime);
  }

  /**
   * 转义正则表达式特殊字符
   * @param str 原始字符串
   * @returns 转义后的字符串
   */
  private escapeRegExp(str: string): string {
    return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  }
}

// 单例实例，方便全局使用
export const cidProcessor = new CidProcessor();
