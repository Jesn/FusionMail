import { useState, useEffect } from 'react';
import { toast } from 'react-hot-toast';
import * as emailListService from '../services/emailListService';
import type { EmailList, AddEmailListRequest } from '../services/emailListService';

export const useEmailList = (type: 'whitelist' | 'blacklist') => {
  const [lists, setLists] = useState<EmailList[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);

  const fetchLists = async () => {
    setIsLoading(true);
    try {
      const response =
        type === 'whitelist'
          ? await emailListService.getWhitelist(page, pageSize)
          : await emailListService.getBlacklist(page, pageSize);

      setLists(response.data);
      setTotal(response.total);
    } catch (error: any) {
      console.error(`Failed to fetch ${type}:`, error);
      toast.error(`获取${type === 'whitelist' ? '白名单' : '黑名单'}失败`);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchLists();
  }, [type, page]);

  const addToList = async (data: AddEmailListRequest) => {
    try {
      if (type === 'whitelist') {
        await emailListService.addToWhitelist(data);
        toast.success('已添加到白名单');
      } else {
        await emailListService.addToBlacklist(data);
        toast.success('已添加到黑名单');
      }
      fetchLists();
    } catch (error: any) {
      console.error(`Failed to add to ${type}:`, error);
      const message = error.response?.data?.error || '添加失败';
      toast.error(message);
      throw error;
    }
  };

  const deleteFromList = async (id: number) => {
    try {
      if (type === 'whitelist') {
        await emailListService.deleteFromWhitelist(id);
        toast.success('已从白名单删除');
      } else {
        await emailListService.deleteFromBlacklist(id);
        toast.success('已从黑名单删除');
      }
      fetchLists();
    } catch (error: any) {
      console.error(`Failed to delete from ${type}:`, error);
      toast.error('删除失败');
      throw error;
    }
  };

  return {
    lists,
    isLoading,
    total,
    page,
    pageSize,
    setPage,
    addToList,
    deleteFromList,
    refresh: fetchLists,
  };
};
