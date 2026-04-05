import { useEffect, useState } from 'react';
import { Save, FolderOpen, Globe, Power, PowerOff, Loader2, Timer, Terminal, Shield, ShieldCheck, Plug, Monitor } from 'lucide-react';
import { useSettingsStore } from '../stores/settingsStore';
import { useAppStore } from '../stores/appStore';
import { GetSettings, UpdateSetting, LaunchBrowser, DisconnectBrowser, GetBrowserStatus, GetBrowserInfo } from '../../wailsjs/go/main/App';
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime';
import { toast } from '../components/ui/Toast';
import type { BrowserStatus, BrowserInfo } from '../types';

const STATUS_STYLES: Record<BrowserStatus, { dot: string; text: string; label: string }> = {
  disconnected: { dot: 'bg-text-muted', text: 'text-text-muted', label: 'Chưa kết nối' },
  connecting:   { dot: 'bg-warning animate-pulse', text: 'text-warning', label: 'Đang kết nối...' },
  connected:    { dot: 'bg-success', text: 'text-success', label: 'Đã kết nối' },
  error:        { dot: 'bg-danger', text: 'text-danger', label: 'Lỗi' },
};

const SETTING_FIELDS = [
  { key: 'chrome_path', label: 'Đường dẫn Chrome', icon: Globe, placeholder: 'Tự động phát hiện' },
  { key: 'user_data_dir', label: 'Thư mục dữ liệu Chrome', icon: FolderOpen, placeholder: 'Mặc định' },
  { key: 'download_folder', label: 'Thư mục tải về', icon: FolderOpen, placeholder: 'Mặc định' },
  { key: 'debug_port', label: 'Debug Port', icon: Terminal, placeholder: '9222' },
  { key: 'delay_between_tasks', label: 'Thời gian chờ giữa các tác vụ (giây)', icon: Timer, placeholder: '5' },
] as const;

export function SettingsPage() {
  const { settings, setSettings, updateSetting } = useSettingsStore();
  const { browserStatus, setBrowserStatus } = useAppStore();
  const [launching, setLaunching] = useState(false);
  const [browserInfo, setBrowserInfo] = useState<BrowserInfo | null>(null);
  const [localSettings, setLocalSettings] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    GetSettings().then(setSettings);
    GetBrowserStatus().then((s) => setBrowserStatus(s as BrowserStatus));
    GetBrowserInfo().then((info) => { if (info) setBrowserInfo(info as any); });
    EventsOn('browser:status', (s: string) => { setBrowserStatus(s as BrowserStatus); setLaunching(false); });
    EventsOn('browser:info', (info: any) => { setBrowserInfo(info); });
    return () => { EventsOff('browser:status'); EventsOff('browser:info'); };
  }, []);

  useEffect(() => {
    setLocalSettings({ ...settings });
  }, [settings]);

  const handleLaunch = async () => {
    setLaunching(true);
    try {
      await LaunchBrowser();
      toast('success', 'Chrome đã kết nối!');
      GetBrowserInfo().then((info) => { if (info) setBrowserInfo(info as any); });
    } catch (err) { toast('error', `${err}`); setLaunching(false); }
  };

  const handleDisconnect = async () => {
    try {
      await DisconnectBrowser();
      toast('info', 'Đã ngắt kết nối');
      setBrowserInfo(null);
    } catch (err) { toast('error', `${err}`); }
  };

  const hasChanges = SETTING_FIELDS.some(({ key }) => (localSettings[key] || '') !== (settings[key] || ''));

  const handleSaveAll = async () => {
    setSaving(true);
    try {
      for (const { key } of SETTING_FIELDS) {
        if ((localSettings[key] || '') !== (settings[key] || '')) {
          await UpdateSetting(key, localSettings[key] || '');
          updateSetting(key, localSettings[key] || '');
        }
      }
      toast('success', 'Đã lưu cài đặt');
    } catch (err) { toast('error', `${err}`); }
    setSaving(false);
  };

  const statusInfo = STATUS_STYLES[browserStatus] || STATUS_STYLES.disconnected;

  return (
    <div className="max-w-2xl h-full overflow-y-auto">
      <h1 className="text-lg font-semibold mb-5">Cài đặt</h1>

      {/* Chrome Connection */}
      <div className="card p-5 mb-5 animate-list-item">
        <h2 className="text-sm font-semibold mb-4 flex items-center gap-2">
          <Globe size={15} /> Trình duyệt Chrome
        </h2>
        <div className="flex items-center gap-4 mb-4">
          <div className="flex items-center gap-2">
            <div className={`w-2 h-2 rounded-full ${statusInfo.dot}`} />
            <span className={`text-sm font-medium ${statusInfo.text}`}>{statusInfo.label}</span>
          </div>
          <div className="flex-1" />
          {browserStatus !== 'connected' ? (
            <button onClick={handleLaunch} disabled={launching} className="btn-primary">
              {launching ? <Loader2 size={15} className="animate-spin" /> : <Power size={15} />}
              {launching ? 'Đang mở...' : 'Mở Chrome'}
            </button>
          ) : (
            <button onClick={handleDisconnect} className="btn-danger" aria-label="Ngắt kết nối Chrome">
              <PowerOff size={15} /> Ngắt kết nối
            </button>
          )}
        </div>

        {browserStatus === 'connected' && browserInfo ? (
          <div className="space-y-3">
            <div className="p-3 bg-bg-tertiary rounded-lg space-y-2">
              <div className="flex items-center gap-2 mb-2">
                <Monitor size={13} className="text-accent" />
                <span className="text-xs font-medium text-text-secondary">Thông tin kết nối</span>
              </div>
              <div className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1.5 text-xs select-text">
                <span className="text-text-muted">Phiên bản:</span>
                <span className="text-text-primary font-mono cursor-text">{browserInfo.version || 'N/A'}</span>
                <span className="text-text-muted">Đường dẫn:</span>
                <span className="text-text-primary font-mono text-[11px] break-all cursor-text">{browserInfo.chromePath || 'N/A'}</span>
                <span className="text-text-muted">Profile:</span>
                <span className="text-text-primary font-mono text-[11px] break-all cursor-text">{browserInfo.profilePath || 'N/A'}</span>
                <span className="text-text-muted">Debug Port:</span>
                <span className="text-text-primary font-mono cursor-text">{browserInfo.debugPort}</span>
              </div>
            </div>

            <div className="p-3 bg-bg-tertiary rounded-lg">
              <div className="flex items-center gap-2 mb-2">
                <Plug size={13} className="text-accent" />
                <span className="text-xs font-medium text-text-secondary">WebSocket</span>
              </div>
              <p className="text-[11px] font-mono text-accent break-all select-text cursor-text">{browserInfo.webSocketURL || 'N/A'}</p>
            </div>

            <div className="p-3 bg-bg-tertiary rounded-lg">
              <div className="flex items-center gap-2 mb-2">
                {browserInfo.stealth ? <ShieldCheck size={13} className="text-success" /> : <Shield size={13} className="text-danger" />}
                <span className="text-xs font-medium text-text-secondary">Anti-Detection (Stealth)</span>
                <span className={`ml-auto text-[10px] font-medium px-1.5 py-0.5 rounded ${browserInfo.stealth ? 'bg-success-subtle text-success' : 'bg-danger-subtle text-danger'}`}>
                  {browserInfo.stealth ? 'ACTIVE' : 'INACTIVE'}
                </span>
              </div>
              {browserInfo.stealth && browserInfo.stealthMods?.length > 0 && (
                <div className="space-y-1">
                  {browserInfo.stealthMods.map((mod, i) => (
                    <div key={i} className="flex items-center gap-2 text-[11px]">
                      <div className="w-1 h-1 rounded-full bg-success shrink-0" />
                      <span className="text-text-muted font-mono select-text cursor-text">{mod}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        ) : (
          <div className="p-3 bg-bg-tertiary rounded-lg">
            <p className="text-xs text-text-muted leading-relaxed">
              Bấm "Mở Chrome" để mở trình duyệt. Đăng nhập một lần, phiên làm việc sẽ được lưu cho lần sau.
            </p>
          </div>
        )}
      </div>

      {/* Fields */}
      <div className="space-y-3">
        {SETTING_FIELDS.map(({ key, label, icon: Icon, placeholder }, index) => (
          <div key={key} className="card p-4 animate-list-item" style={{ animationDelay: `${(index + 1) * 60}ms` }}>
            <label htmlFor={`setting-${key}`} className="flex items-center gap-2 text-xs font-medium text-text-secondary mb-2">
              <Icon size={13} /> {label}
            </label>
            <input
              id={`setting-${key}`}
              value={localSettings[key] || ''}
              onChange={(e) => setLocalSettings(prev => ({ ...prev, [key]: e.target.value }))}
              placeholder={placeholder}
              className="input-field w-full"
            />
          </div>
        ))}
        <button
          onClick={handleSaveAll}
          disabled={!hasChanges || saving}
          className="btn-primary w-full py-2.5 mt-2 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {saving ? <Loader2 size={15} className="animate-spin" /> : <Save size={15} />}
          {saving ? 'Đang lưu...' : 'Lưu cài đặt'}
        </button>
      </div>

      {/* Debug */}
      <div className="card p-4 mt-5 animate-list-item" style={{ animationDelay: '400ms' }}>
        <h2 className="text-xs font-medium text-text-secondary mb-2">Thông tin gỡ lỗi</h2>
        <div className="text-xs text-text-muted space-y-1">
          <p>Default Model: veo_3_1_t2v_lite</p>
          <p>API: aisandbox-pa.googleapis.com/v1</p>
          <p>Trình duyệt: <span className={statusInfo.text}>{statusInfo.label}</span></p>
        </div>
      </div>
    </div>
  );
}
