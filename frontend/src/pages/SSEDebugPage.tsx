import { useState, useEffect, useRef } from 'react';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';

/**
 * SSE 调试页面
 * 用于测试和调试 SSE 连接
 */
export const SSEDebugPage = () => {
  const [logs, setLogs] = useState<string[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [connectionStatus, setConnectionStatus] = useState<'disconnected' | 'connecting' | 'connected' | 'error'>('disconnected');
  const [cookies, setCookies] = useState<string>('');
  const eventSourceRef = useRef<EventSource | null>(null);

  const addLog = (message: string) => {
    const timestamp = new Date().toLocaleTimeString();
    setLogs(prev => [...prev, `[${timestamp}] ${message}`]);
  };

  const checkCookies = () => {
    const allCookies = document.cookie;
    setCookies(allCookies);
    addLog(`当前 Cookie: ${allCookies || '(无)'}`);
    
    if (allCookies.includes('fm_session')) {
      addLog('✅ 找到 fm_session Cookie');
    } else {
      addLog('❌ 未找到 fm_session Cookie');
    }
  };

  const connectSSE = () => {
    if (eventSourceRef.current) {
      addLog('⚠️ 已存在连接，先关闭旧连接');
      eventSourceRef.current.close();
    }

    const API_BASE = import.meta.env.VITE_API_BASE_URL || '';
    const url = `${API_BASE}/api/v1/events`;
    
    addLog(`开始连接 SSE: ${url}`);
    setConnectionStatus('connecting');

    try {
      const es = new EventSource(url, { withCredentials: true });
      eventSourceRef.current = es;

      es.addEventListener('open', () => {
        addLog('✅ SSE 连接已建立');
        setIsConnected(true);
        setConnectionStatus('connected');
      });

      es.addEventListener('error', (e) => {
        addLog(`❌ SSE 连接错误: ${JSON.stringify(e)}`);
        setConnectionStatus('error');
        
        // 检查连接状态
        if (es.readyState === EventSource.CLOSED) {
          addLog('连接已关闭');
          setIsConnected(false);
        } else if (es.readyState === EventSource.CONNECTING) {
          addLog('正在重连...');
        }
      });

      es.addEventListener('email_counts_maybe_changed', (e) => {
        addLog(`📧 收到事件: email_counts_maybe_changed, 数据: ${(e as MessageEvent).data}`);
      });

      es.addEventListener('ping', (e) => {
        addLog(`💓 收到心跳: ${(e as MessageEvent).data}`);
      });

      es.addEventListener('message', (e) => {
        addLog(`📨 收到消息: ${(e as MessageEvent).data}`);
      });

    } catch (error) {
      addLog(`❌ 连接失败: ${error}`);
      setConnectionStatus('error');
    }
  };

  const disconnectSSE = () => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
      setIsConnected(false);
      setConnectionStatus('disconnected');
      addLog('🔌 SSE 连接已关闭');
    } else {
      addLog('⚠️ 没有活动的连接');
    }
  };

  const clearLogs = () => {
    setLogs([]);
  };

  const testStatsAPI = async () => {
    addLog('测试 /emails/stats 接口...');
    try {
      const response = await fetch('/api/v1/emails/stats', {
        credentials: 'include', // 包含 Cookie
      });
      
      addLog(`响应状态: ${response.status} ${response.statusText}`);
      
      if (response.ok) {
        const data = await response.json();
        addLog(`✅ 响应数据: ${JSON.stringify(data)}`);
      } else {
        const text = await response.text();
        addLog(`❌ 错误响应: ${text}`);
      }
    } catch (error) {
      addLog(`❌ 请求失败: ${error}`);
    }
  };

  useEffect(() => {
    checkCookies();
    
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
      }
    };
  }, []);

  const getStatusBadge = () => {
    switch (connectionStatus) {
      case 'connected':
        return <Badge className="bg-green-500">已连接</Badge>;
      case 'connecting':
        return <Badge className="bg-yellow-500">连接中...</Badge>;
      case 'error':
        return <Badge className="bg-red-500">错误</Badge>;
      default:
        return <Badge variant="secondary">未连接</Badge>;
    }
  };

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div>
        <h1 className="text-3xl font-bold">SSE 调试工具</h1>
        <p className="text-muted-foreground mt-2">
          用于测试和调试 Server-Sent Events 连接
        </p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* 连接控制 */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center justify-between">
              <span>连接控制</span>
              {getStatusBadge()}
            </CardTitle>
            <CardDescription>
              管理 SSE 连接状态
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex gap-2">
              <Button 
                onClick={connectSSE} 
                disabled={isConnected}
                className="flex-1"
              >
                连接 SSE
              </Button>
              <Button 
                onClick={disconnectSSE} 
                disabled={!isConnected}
                variant="destructive"
                className="flex-1"
              >
                断开连接
              </Button>
            </div>
            
            <Button 
              onClick={checkCookies} 
              variant="outline"
              className="w-full"
            >
              检查 Cookie
            </Button>
            
            <Button 
              onClick={testStatsAPI} 
              variant="outline"
              className="w-full"
            >
              测试 /emails/stats 接口
            </Button>
          </CardContent>
        </Card>

        {/* Cookie 信息 */}
        <Card>
          <CardHeader>
            <CardTitle>Cookie 信息</CardTitle>
            <CardDescription>
              当前浏览器的 Cookie 状态
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="bg-muted p-4 rounded-md font-mono text-sm break-all">
              {cookies || '(无 Cookie)'}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 日志 */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center justify-between">
            <span>事件日志</span>
            <Button onClick={clearLogs} variant="outline" size="sm">
              清空日志
            </Button>
          </CardTitle>
          <CardDescription>
            实时显示 SSE 事件和调试信息
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="bg-black text-green-400 p-4 rounded-md font-mono text-sm h-96 overflow-y-auto">
            {logs.length === 0 ? (
              <div className="text-gray-500">暂无日志</div>
            ) : (
              logs.map((log, index) => (
                <div key={index} className="mb-1">
                  {log}
                </div>
              ))
            )}
          </div>
        </CardContent>
      </Card>

      {/* 使用说明 */}
      <Card>
        <CardHeader>
          <CardTitle>使用说明</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          <p>1. 首先点击"检查 Cookie"确认 fm_session Cookie 存在</p>
          <p>2. 点击"连接 SSE"建立 Server-Sent Events 连接</p>
          <p>3. 观察日志中的连接状态和事件</p>
          <p>4. 可以点击"测试 /emails/stats 接口"验证 API 调用</p>
          <p>5. 在其他页面进行操作（如标记邮件为已读）时，应该能看到 SSE 事件</p>
        </CardContent>
      </Card>
    </div>
  );
};

