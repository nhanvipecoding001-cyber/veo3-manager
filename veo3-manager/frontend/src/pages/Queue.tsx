import { useState, useEffect } from 'react';
import { Play, Pause, Square, Plus, Upload, Loader2 } from 'lucide-react';
import { useQueueStore } from '../stores/queueStore';
import { useAppStore } from '../stores/appStore';
import {
  CreateTasksBatch, ListTasks, DeleteTask,
  StartQueue, PauseQueue, ResumeQueue, StopQueue,
  LaunchBrowser, GetBrowserStatus, GetQueueState,
} from '../../wailsjs/go/main/App';
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime';
import { toast } from '../components/ui/Toast';
import { TaskListSkeleton } from '../components/ui/Skeleton';

export function Queue() {
  const { tasks, queueState, currentTaskId, setTasks, addTasks, updateTask, removeTask, setQueueState, setCurrentTaskId, setStats } = useQueueStore();
  const { browserStatus, setBrowserStatus } = useAppStore();
  const [promptText, setPromptText] = useState('');
  const [aspectRatio, setAspectRatio] = useState('16:9');
  const [model, setModel] = useState('veo_3_1_t2v_lite');
  const [outputCount, setOutputCount] = useState(1);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      ListTasks({ status: '', search: '', limit: 100, offset: 0 }).then((t) => { if (t) setTasks(t); }),
      GetBrowserStatus().then(setBrowserStatus as any),
      GetQueueState().then((s) => setQueueState(s as any)),
    ]).finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    const handlers: [string, (...args: any[]) => void][] = [
      ['browser:status', (s: string) => setBrowserStatus(s as any)],
      ['queue:state', (s: string) => setQueueState(s as any)],
      ['task:started', (d: any) => { setCurrentTaskId(d.taskId); updateTask(d.taskId, { status: 'processing' }); }],
      ['task:progress', (d: any) => { updateTask(d.taskId, { status: d.status }); }],
      ['task:completed', (d: any) => { updateTask(d.taskId, { status: 'completed' }); setCurrentTaskId(null); toast('success', 'Tác vụ hoàn thành!'); }],
      ['task:failed', (d: any) => { updateTask(d.taskId, { status: 'failed', errorMessage: d.error }); setCurrentTaskId(null); toast('error', `Tác vụ thất bại: ${d.error}`); }],
      ['queue:stats', (s: any) => setStats(s)],
    ];
    handlers.forEach(([e, h]) => EventsOn(e, h));
    return () => { handlers.forEach(([e]) => EventsOff(e)); };
  }, []);

  const handleAddPrompts = async () => {
    const prompts = promptText.split('\n').filter((p) => p.trim());
    if (!prompts.length) return;
    try {
      const t = await CreateTasksBatch(prompts, aspectRatio, model, outputCount);
      if (t) { addTasks(t); setPromptText(''); toast('success', `Đã thêm ${t.length} tác vụ`); }
    } catch (err) { toast('error', `${err}`); }
  };

  const handleStart = async () => {
    if (browserStatus !== 'connected') {
      try { toast('info', 'Đang mở trình duyệt...'); await LaunchBrowser(); } catch (err) { toast('error', `${err}`); return; }
    }
    try { await StartQueue(); toast('info', 'Hàng đợi đã bắt đầu'); } catch (err) { toast('error', `${err}`); }
  };

  const handlePause = () => {
    if (queueState === 'paused') { ResumeQueue(); toast('info', 'Đã tiếp tục'); }
    else { PauseQueue(); toast('info', 'Đã tạm dừng'); }
  };

  const handleStop = () => { StopQueue(); toast('info', 'Đang dừng...'); };

  const badge = (status: string) => {
    const styles: Record<string, string> = {
      pending: 'bg-text-muted/20 text-text-muted',
      processing: 'bg-accent-subtle text-accent',
      polling: 'bg-warning-subtle text-warning',
      downloading: 'bg-accent-subtle text-accent-hover',
      completed: 'bg-success-subtle text-success',
      failed: 'bg-danger-subtle text-danger',
      cancelled: 'bg-text-muted/20 text-text-muted',
    };
    const active = status === 'processing' || status === 'polling' || status === 'downloading';
    return (
      <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${styles[status] || ''}`}>
        {active && <Loader2 size={10} className="animate-spin" />}
        {status}
      </span>
    );
  };

  return (
    <div className="flex flex-col gap-4 h-full">
      {/* Toolbar */}
      <div className="card p-3 flex items-center gap-4 shrink-0 flex-wrap">
        <div className="flex items-center gap-2">
          <label htmlFor="aspect-ratio" className="text-xs text-text-muted">Tỷ lệ</label>
          <select id="aspect-ratio" value={aspectRatio} onChange={(e) => setAspectRatio(e.target.value)} className="input-field py-1">
            <option value="16:9">16:9</option>
            <option value="9:16">9:16</option>
          </select>
        </div>
        <div className="flex items-center gap-2">
          <label htmlFor="model-select" className="text-xs text-text-muted">Model</label>
          <select id="model-select" value={model} onChange={(e) => setModel(e.target.value)} className="input-field py-1">
            <option value="veo_3_1_t2v_lite">Lite</option>
            <option value="veo_3_1_t2v_fast">Fast</option>
            <option value="veo_3_1_t2v_quality">Quality</option>
          </select>
        </div>
        <div className="flex items-center gap-2">
          <label htmlFor="output-count" className="text-xs text-text-muted">Số video</label>
          <select id="output-count" value={outputCount} onChange={(e) => setOutputCount(Number(e.target.value))} className="input-field py-1">
            {[1, 2, 3, 4].map((n) => <option key={n} value={n}>{n}</option>)}
          </select>
        </div>
        <div className="flex-1" />
        <span className="text-xs text-text-muted uppercase tracking-wider font-medium">{queueState}</span>
        <div className="flex items-center gap-1">
          <button onClick={handleStart} disabled={queueState === 'running'} title="Bắt đầu" aria-label="Bắt đầu hàng đợi" className="p-2 rounded-lg bg-success-subtle text-success hover:bg-success/20 transition-all active:scale-95 disabled:opacity-30 disabled:active:scale-100">
            <Play size={15} />
          </button>
          <button onClick={handlePause} disabled={queueState !== 'running' && queueState !== 'paused'} title={queueState === 'paused' ? 'Tiếp tục' : 'Tạm dừng'} aria-label={queueState === 'paused' ? 'Tiếp tục' : 'Tạm dừng'} className="p-2 rounded-lg bg-warning-subtle text-warning hover:bg-warning/20 transition-all active:scale-95 disabled:opacity-30 disabled:active:scale-100">
            <Pause size={15} />
          </button>
          <button onClick={handleStop} disabled={queueState === 'idle'} title="Dừng" aria-label="Dừng hàng đợi" className="p-2 rounded-lg bg-danger-subtle text-danger hover:bg-danger/20 transition-all active:scale-95 disabled:opacity-30 disabled:active:scale-100">
            <Square size={15} />
          </button>
        </div>
      </div>

      {/* Prompt input */}
      <div className="card p-3 shrink-0">
        <label htmlFor="prompt-input" className="sr-only">Nhập prompt</label>
        <textarea
          id="prompt-input"
          value={promptText}
          onChange={(e) => setPromptText(e.target.value)}
          placeholder="Nhập prompt (mỗi dòng một prompt)..."
          className="input-field w-full h-24 resize-none p-3"
        />
        <div className="flex justify-end mt-2">
          <button onClick={handleAddPrompts} disabled={!promptText.trim()} className="btn-primary">
            <Plus size={14} /> Thêm vào hàng đợi
          </button>
        </div>
      </div>

      {/* Task list */}
      <div className="flex-1 min-h-0 overflow-y-auto">
        {loading ? (
          <TaskListSkeleton count={4} />
        ) : tasks.length === 0 ? (
          <div className="flex flex-col items-center justify-center text-text-muted py-16">
            <Upload size={36} className="mb-3 opacity-30" />
            <p className="text-sm">Chưa có tác vụ. Thêm prompt ở trên.</p>
          </div>
        ) : (
          <div className="space-y-1.5">
            {tasks.map((task, index) => (
              <div
                key={task.id}
                className={`card card-interactive p-3 flex items-center gap-3 animate-list-item ${
                  currentTaskId === task.id ? '!border-accent' : ''
                }`}
                style={{ animationDelay: `${index * 40}ms` }}
              >
                <div className="flex-1 min-w-0">
                  <p className="text-sm truncate">{task.prompt}</p>
                  <p className="text-xs text-text-muted mt-0.5">
                    {task.aspectRatio} &middot; {task.outputCount} video(s)
                    {task.errorMessage && <span className="text-danger ml-2">{task.errorMessage}</span>}
                  </p>
                </div>
                {badge(task.status)}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
