import { Minus, X, Maximize2, Sun, Moon } from 'lucide-react';
import { WindowMinimise, WindowToggleMaximise, WindowClose } from '../../../wailsjs/go/main/App';
import { useTheme } from '../ThemeProvider';

export function TitleBar() {
  const { theme, toggleTheme } = useTheme();

  return (
    <div
      className="flex items-center justify-between h-10 bg-bg-secondary border-b border-border shrink-0 select-none"
      style={{ '--wails-draggable': 'drag' } as React.CSSProperties}
    >
      {/* Spacer left */}
      <div className="w-32" />

      {/* Center title */}
      <div className="flex items-center gap-2.5">
        <div className="w-2.5 h-2.5 rounded-full bg-accent" />
        <span className="text-xs font-semibold text-text-secondary tracking-wider uppercase">
          Thanh Nhàn VEO 3
        </span>
      </div>

      {/* Window controls */}
      <div className="flex h-full items-center" style={{ '--wails-draggable': 'no-drag' } as React.CSSProperties}>
        <button
          onClick={toggleTheme}
          className="w-10 h-full flex items-center justify-center hover:bg-bg-tertiary transition-colors text-text-muted hover:text-text-primary"
          aria-label={theme === 'dark' ? 'Chuyển sang sáng' : 'Chuyển sang tối'}
          title={theme === 'dark' ? 'Chế độ sáng' : 'Chế độ tối'}
        >
          {theme === 'dark' ? <Sun size={14} /> : <Moon size={14} />}
        </button>
        <button
          onClick={() => WindowMinimise()}
          className="w-11 h-full flex items-center justify-center hover:bg-bg-tertiary transition-colors text-text-muted hover:text-text-primary"
          aria-label="Thu nhỏ"
        >
          <Minus size={14} />
        </button>
        <button
          onClick={() => WindowToggleMaximise()}
          className="w-11 h-full flex items-center justify-center hover:bg-bg-tertiary transition-colors text-text-muted hover:text-text-primary"
          aria-label="Phóng to"
        >
          <Maximize2 size={13} />
        </button>
        <button
          onClick={() => WindowClose()}
          className="w-11 h-full flex items-center justify-center hover:bg-danger transition-colors text-text-muted hover:text-white"
          aria-label="Đóng"
        >
          <X size={14} />
        </button>
      </div>
    </div>
  );
}
