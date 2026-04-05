import { useState, useEffect } from 'react';
import { Search, ChevronLeft, ChevronRight, Trash2 } from 'lucide-react';
import { ListTasks, DeleteTask, GetVideoData } from '../../wailsjs/go/main/App';
import { toast } from '../components/ui/Toast';
import { TableSkeleton, VideoCardSkeleton } from '../components/ui/Skeleton';
import type { Task } from '../types';

export function HistoryPage() {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [selectedTask, setSelectedTask] = useState<Task | null>(null);
  const [videoIndex, setVideoIndex] = useState(0);
  const [videoSrc, setVideoSrc] = useState<string | null>(null);
  const [videoLoading, setVideoLoading] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    ListTasks({ status: statusFilter, search, limit: 50, offset: 0 }).then((t) => {
      if (t) setTasks(t);
    }).finally(() => setLoading(false));
  }, [search, statusFilter]);

  useEffect(() => {
    if (selectedTask?.videoPaths?.length && selectedTask.videoPaths[videoIndex]) {
      setVideoLoading(true);
      setVideoSrc(null);
      GetVideoData(selectedTask.videoPaths[videoIndex]).then((data) => {
        setVideoSrc(data);
        setVideoLoading(false);
      }).catch(() => setVideoLoading(false));
    } else {
      setVideoSrc(null);
    }
  }, [selectedTask?.id, videoIndex]);

  const handleDelete = async (id: string) => {
    try {
      await DeleteTask(id);
      setTasks((prev) => prev.filter((t) => t.id !== id));
      if (selectedTask?.id === id) setSelectedTask(null);
      toast('success', 'Đã xóa');
    } catch (err) {
      toast('error', `${err}`);
    }
  };

  const statusColor = (s: string) =>
    s === 'completed' ? 'text-success' : s === 'failed' ? 'text-danger' : 'text-text-muted';

  const formatModel = (model: string) => {
    if (model?.includes('lite')) return 'Lite';
    if (model?.includes('fast')) return 'Fast';
    if (model?.includes('quality')) return 'Quality';
    return model || 'Lite';
  };

  return (
    <div className="flex flex-col gap-4 h-full">
      {/* Header */}
      <div className="flex items-center gap-3 shrink-0 flex-wrap">
        <h1 className="text-lg font-semibold shrink-0">Lịch sử</h1>
        <div className="flex-1 min-w-[40px]" />
        <div className="relative shrink-0">
          <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted pointer-events-none" />
          <label htmlFor="history-search" className="sr-only">Tìm kiếm prompt</label>
          <input
            id="history-search"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Tìm kiếm prompt..."
            className="input-field w-56"
            style={{ paddingLeft: '2.25rem' }}
          />
        </div>
        <label htmlFor="status-filter" className="sr-only">Lọc trạng thái</label>
        <select id="status-filter" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} className="input-field shrink-0">
          <option value="">Tất cả</option>
          <option value="completed">Hoàn thành</option>
          <option value="failed">Thất bại</option>
          <option value="pending">Đang chờ</option>
        </select>
      </div>

      {/* Content */}
      <div className="flex flex-1 gap-4 min-h-0">
        {/* Table */}
        <div className="flex-1 overflow-y-auto card">
          {loading ? (
            <TableSkeleton rows={6} />
          ) : (
            <table className="w-full text-sm">
              <thead className="text-text-muted text-xs sticky top-0 bg-bg-secondary z-10">
                <tr className="border-b border-border">
                  <th className="text-left p-3 font-medium">Prompt</th>
                  <th className="text-left p-3 w-20 font-medium">Model</th>
                  <th className="text-left p-3 w-16 font-medium">Tỷ lệ</th>
                  <th className="text-left p-3 w-24 font-medium">Trạng thái</th>
                  <th className="text-left p-3 w-16 font-medium">Video</th>
                  <th className="text-left p-3 w-28 font-medium">Ngày</th>
                  <th className="p-3 w-10"><span className="sr-only">Hành động</span></th>
                </tr>
              </thead>
              <tbody>
                {tasks.map((task, index) => (
                  <tr
                    key={task.id}
                    onClick={() => { setSelectedTask(task); setVideoIndex(0); }}
                    className={`cursor-pointer border-b border-border/40 hover:bg-bg-tertiary transition-colors animate-list-item ${
                      selectedTask?.id === task.id ? 'bg-bg-tertiary' : ''
                    }`}
                    style={{ animationDelay: `${index * 30}ms` }}
                    tabIndex={0}
                    role="row"
                    aria-selected={selectedTask?.id === task.id}
                    onKeyDown={(e) => { if (e.key === 'Enter') { setSelectedTask(task); setVideoIndex(0); } }}
                  >
                    <td className="p-3 truncate max-w-xs">{task.prompt}</td>
                    <td className="p-3 text-text-muted text-xs">{formatModel(task.model)}</td>
                    <td className="p-3 text-text-muted text-xs">{task.aspectRatio || '16:9'}</td>
                    <td className="p-3"><span className={`text-xs font-medium ${statusColor(task.status)}`}>{task.status}</span></td>
                    <td className="p-3 text-text-muted">{task.videoPaths?.length || 0}</td>
                    <td className="p-3 text-text-muted text-xs">{new Date(task.createdAt).toLocaleDateString()}</td>
                    <td className="p-3">
                      <button
                        onClick={(e) => { e.stopPropagation(); handleDelete(task.id); }}
                        className="p-1 text-text-muted hover:text-danger transition-colors active:scale-90"
                        title="Xóa"
                        aria-label={`Xóa tác vụ: ${task.prompt.slice(0, 30)}`}
                      >
                        <Trash2 size={14} />
                      </button>
                    </td>
                  </tr>
                ))}
                {!loading && tasks.length === 0 && (
                  <tr><td colSpan={7} className="text-center text-text-muted py-12 text-sm">Không tìm thấy tác vụ</td></tr>
                )}
              </tbody>
            </table>
          )}
        </div>

        {/* Video preview */}
        {selectedTask && selectedTask.videoPaths?.length > 0 && (
          <div className="w-72 xl:w-80 card p-3 flex flex-col gap-2 shrink-0 animate-list-item">
            {videoLoading ? (
              <div className="w-full rounded-lg aspect-video skeleton" />
            ) : videoSrc ? (
              <video
                key={selectedTask.videoPaths[videoIndex]}
                src={videoSrc}
                controls autoPlay muted
                className="w-full rounded-lg bg-black aspect-video"
              />
            ) : (
              <div className="w-full rounded-lg bg-bg-tertiary aspect-video flex items-center justify-center text-text-muted text-sm">
                Không tải được video
              </div>
            )}
            {selectedTask.videoPaths.length > 1 && (
              <div className="flex items-center justify-center gap-3">
                <button onClick={() => setVideoIndex((i) => Math.max(0, i - 1))} disabled={videoIndex === 0} className="p-1 text-text-muted hover:text-text-primary disabled:opacity-25 active:scale-90 transition-all" aria-label="Video trước">
                  <ChevronLeft size={16} />
                </button>
                <span className="text-xs text-text-muted">{videoIndex + 1} / {selectedTask.videoPaths.length}</span>
                <button onClick={() => setVideoIndex((i) => Math.min(selectedTask.videoPaths.length - 1, i + 1))} disabled={videoIndex === selectedTask.videoPaths.length - 1} className="p-1 text-text-muted hover:text-text-primary disabled:opacity-25 active:scale-90 transition-all" aria-label="Video sau">
                  <ChevronRight size={16} />
                </button>
              </div>
            )}
            <p className="text-xs text-text-secondary line-clamp-3 mt-1">{selectedTask.prompt}</p>
          </div>
        )}
      </div>
    </div>
  );
}
