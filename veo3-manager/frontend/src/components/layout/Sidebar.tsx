import { useEffect } from 'react';
import { LayoutDashboard, ListVideo, History, Settings, Globe, Loader2 } from 'lucide-react';
import { useAppStore } from '../../stores/appStore';
import { LaunchBrowser, DisconnectBrowser, GetBrowserStatus } from '../../../wailsjs/go/main/App';
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime';
import { toast } from '../ui/Toast';
import type { Page, BrowserStatus } from '../../types';

const navItems: { id: Page; label: string; icon: React.ElementType }[] = [
  { id: 'dashboard', label: 'Tổng quan', icon: LayoutDashboard },
  { id: 'queue', label: 'Hàng đợi', icon: ListVideo },
  { id: 'history', label: 'Lịch sử', icon: History },
  { id: 'settings', label: 'Cài đặt', icon: Settings },
];

const statusDot: Record<BrowserStatus, string> = {
  disconnected: 'bg-text-muted',
  connecting: 'bg-warning animate-pulse',
  connected: 'bg-success',
  error: 'bg-danger',
};

const statusLabel: Record<BrowserStatus, string> = {
  disconnected: 'Kết nối',
  connecting: 'Đang kết nối',
  connected: 'Đã kết nối',
  error: 'Lỗi',
};

interface SidebarProps {
  collapsed?: boolean;
  onClose?: () => void;
}

export function Sidebar({ collapsed = false, onClose }: SidebarProps) {
  const { currentPage, setPage, browserStatus, setBrowserStatus } = useAppStore();

  useEffect(() => {
    GetBrowserStatus().then((s) => setBrowserStatus(s as BrowserStatus));
    EventsOn('browser:status', (s: string) => setBrowserStatus(s as BrowserStatus));
    return () => { EventsOff('browser:status'); };
  }, []);

  const handleBrowserClick = async () => {
    if (browserStatus === 'connecting') return;
    if (browserStatus === 'connected') {
      try {
        await DisconnectBrowser();
        toast('info', 'Đã ngắt kết nối Chrome');
      } catch (err) {
        toast('error', `${err}`);
      }
    } else {
      try {
        toast('info', 'Đang mở Chrome...');
        await LaunchBrowser();
        toast('success', 'Chrome đã kết nối!');
      } catch (err) {
        toast('error', `Lỗi: ${err}`);
      }
    }
  };

  const handleNavClick = (id: Page) => {
    setPage(id);
    onClose?.();
  };

  return (
    <aside
      className={`flex flex-col bg-bg-secondary border-r border-border shrink-0 transition-all duration-200 ${
        collapsed ? 'w-14' : 'w-[180px]'
      }`}
      role="navigation"
      aria-label="Menu chính"
    >
      <nav className="flex flex-col gap-1 pt-3 px-2 flex-1">
        {navItems.map(({ id, label, icon: Icon }) => (
          <button
            key={id}
            onClick={() => handleNavClick(id)}
            className={`flex items-center gap-3 h-10 rounded-lg transition-all text-sm font-medium ${
              collapsed ? 'justify-center px-0' : 'px-3'
            } ${
              currentPage === id
                ? 'bg-accent text-white shadow-sm'
                : 'text-text-muted hover:text-text-primary hover:bg-bg-tertiary'
            }`}
            title={label}
            aria-label={label}
            aria-current={currentPage === id ? 'page' : undefined}
          >
            <Icon size={17} className="shrink-0" />
            {!collapsed && <span>{label}</span>}
          </button>
        ))}
      </nav>

      {/* Browser toggle */}
      <div className="px-2 pb-3">
        <button
          onClick={handleBrowserClick}
          className={`w-full flex items-center gap-3 h-10 rounded-lg transition-all text-sm font-medium ${
            collapsed ? 'justify-center px-0' : 'px-3'
          } ${
            browserStatus === 'connected'
              ? 'text-success hover:bg-bg-tertiary'
              : 'text-text-muted hover:bg-bg-tertiary hover:text-text-primary'
          }`}
          title={browserStatus === 'connected' ? 'Chrome đã kết nối' : 'Mở Chrome'}
          aria-label={`Chrome: ${statusLabel[browserStatus]}`}
        >
          {browserStatus === 'connecting' ? (
            <Loader2 size={16} className="animate-spin text-warning shrink-0" />
          ) : (
            <Globe size={16} className="shrink-0" />
          )}
          {!collapsed && <span>{statusLabel[browserStatus]}</span>}
          <div className={`w-2 h-2 rounded-full ${collapsed ? '' : 'ml-auto'} ${statusDot[browserStatus]}`} />
        </button>
      </div>
    </aside>
  );
}
