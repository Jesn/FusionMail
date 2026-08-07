import { Button } from '../ui/button';
import { ChevronLeft, ChevronRight } from 'lucide-react';

interface EmailPaginationProps {
  page: number;
  totalPages: number;
  total: number;
  onPrev: () => void;
  onNext: () => void;
}

export const EmailPagination = ({
  page,
  totalPages,
  total,
  onPrev,
  onNext,
}: EmailPaginationProps) => {
  if (totalPages <= 1) return null;

  return (
    <div className="flex items-center justify-between border-t bg-background px-4 py-2">
      <div className="text-sm text-muted-foreground">
        第 {page} 页，共 {totalPages} 页 · 总计 {total} 封邮件
      </div>
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          onClick={onPrev}
          disabled={page === 1}
        >
          <ChevronLeft className="h-4 w-4" />
          上一页
        </Button>
        <Button
          variant="outline"
          size="sm"
          onClick={onNext}
          disabled={page === totalPages}
        >
          下一页
          <ChevronRight className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
};