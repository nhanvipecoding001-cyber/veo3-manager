import { useState, useEffect } from 'react';
import { Menu, X } from 'lucide-react';
import { TitleBar } from './TitleBar';
import { Sidebar } from './Sidebar';
import { useAppStore } from '../../stores/appStore';
import { Dashboard } from '../../pages/Dashboard';
import { Queue } from '../../pages/Queue';
import { HistoryPage } from '../../pages/History';
import { SettingsPage } from '../../pages/Settings';

const pages = {
  dashboard: Dashboard,
  queue: Queue,
  history: HistoryPage,
  settings: SettingsPage,
};

const BREAKPOINT = 640;

export function AppLayout() {
  const currentPage = useAppStore((s) => s.currentPage);
  const PageComponent = pages[currentPage];
  const [isNarrow, setIsNarrow] = useState(window.innerWidth < BREAKPOINT);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  useEffect(() => {
    const onResize = () => {
      const narrow = window.innerWidth < BREAKPOINT;
      setIsNarrow(narrow);
      if (!narrow) setMobileMenuOpen(false);
    };
    window.addEventListener('resize', onResize);
    return () => window.removeEventListener('resize', onResize);
  }, []);

  return (
    <div className="flex flex-col h-full w-full">
      <TitleBar />
      <div className="flex flex-1 min-h-0 relative">
        {/* Desktop sidebar */}
        {!isNarrow && <Sidebar />}

        {/* Mobile overlay */}
        {isNarrow && mobileMenuOpen && (
          <>
            <div className="fixed inset-0 z-40 bg-black/40" onClick={() => setMobileMenuOpen(false)} />
            <div className="fixed left-0 top-10 bottom-0 z-50">
              <Sidebar onClose={() => setMobileMenuOpen(false)} />
            </div>
          </>
        )}

        {/* Main content */}
        <main className="flex-1 min-w-0 overflow-y-auto p-5">
          {isNarrow && (
            <button
              onClick={() => setMobileMenuOpen(true)}
              className="btn-ghost mb-3 -ml-1"
              aria-label="Mở menu"
            >
              <Menu size={18} />
            </button>
          )}
          <PageComponent />
        </main>
      </div>
    </div>
  );
}
