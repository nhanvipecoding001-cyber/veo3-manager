import { useEffect } from 'react';
import { Video, CheckCircle, Clock, AlertCircle } from 'lucide-react';
import { useQueueStore } from '../stores/queueStore';
import { GetTaskStats } from '../../wailsjs/go/main/App';
import { Skeleton } from '../components/ui/Skeleton';

export function Dashboard() {
  const { stats, setStats } = useQueueStore();

  useEffect(() => {
    GetTaskStats()
      .then(setStats)
      .catch(() => setStats({ total: 0, completed: 0, pending: 0, processing: 0, failed: 0 }));
  }, [setStats]);

  const cards = [
    { label: 'Tổng tác vụ', value: stats?.total ?? 0, icon: Video, color: 'text-accent' },
    { label: 'Hoàn thành', value: stats?.completed ?? 0, icon: CheckCircle, color: 'text-success' },
    { label: 'Đang chờ', value: stats?.pending ?? 0, icon: Clock, color: 'text-warning' },
    { label: 'Thất bại', value: stats?.failed ?? 0, icon: AlertCircle, color: 'text-danger' },
  ];

  return (
    <div className="h-full">
      <h1 className="text-lg font-semibold mb-5">Tổng quan</h1>
      <div className="grid grid-cols-2 xl:grid-cols-4 gap-4">
        {!stats ? (
          Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="card p-4 flex items-center gap-3">
              <Skeleton className="w-10 h-10 rounded-lg" />
              <div className="space-y-2 flex-1">
                <Skeleton className="h-6 w-12" />
                <Skeleton className="h-3 w-16" />
              </div>
            </div>
          ))
        ) : (
          cards.map(({ label, value, icon: Icon, color }, index) => (
            <div
              key={label}
              className="card card-interactive p-4 flex items-center gap-3 animate-list-item"
              style={{ animationDelay: `${index * 80}ms` }}
            >
              <div className={`p-2.5 rounded-lg bg-bg-tertiary ${color}`}>
                <Icon size={20} />
              </div>
              <div>
                <p className="text-2xl font-bold leading-tight">{value}</p>
                <p className="text-xs text-text-muted mt-0.5">{label}</p>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
