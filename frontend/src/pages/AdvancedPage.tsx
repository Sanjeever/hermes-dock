import {useState} from 'react';
import {gotoLine, openSearchPanel} from '@codemirror/search';
import {EditorView} from '@codemirror/view';
import {CornerDownRight, FileCode2, Save, Search, Trash2} from 'lucide-react';
import {CodeEditor} from '../components/CodeEditor';

export function AdvancedPage(props: { options: Array<{ value: string; label: string }>; path: string; setPath: (value: string) => void; open: boolean; setOpen: (value: boolean) => void; content: string; setContent: (value: string) => void; status: string; dirty: boolean; busy: boolean; onSave: () => void; onFactoryReset: () => Promise<void>; resetConfirmPhrase: string; hideFactoryReset?: boolean }) {
    const [editorView, setEditorView] = useState<EditorView | null>(null);
    const [resetConfirmText, setResetConfirmText] = useState('');
    const languageLabel = props.path.endsWith('.env') ? '.env' : props.path.endsWith('.md') ? 'Markdown' : 'YAML';
    const resetConfirmed = resetConfirmText === props.resetConfirmPhrase;

    async function factoryReset() {
        if (!resetConfirmed) return;
        await props.onFactoryReset();
        setResetConfirmText('');
    }

    return (
        <section className="advanced-stack">
            <div className="panel">
                <div className="section-head">
                    <div>
                        <p className="eyebrow">高级编辑</p>
                        <h2>{props.open ? props.path : '选择要编辑的文件'}</h2>
                        {!props.open && <p className="setup-subtitle">直接修改原始配置文件。保存后通常需要应用并重建才会生效。</p>}
                    </div>
                    {props.open && <span className={`inline-status ${props.dirty ? 'dirty' : ''}`}>{props.dirty ? '有未保存修改' : props.status}</span>}
                </div>
                {!props.open ? (
                    <div className="advanced-file-grid">
                        {props.options.map((option) => (
                            <button key={option.value} className="advanced-file-card" onClick={() => {
                                props.setPath(option.value);
                                props.setOpen(true);
                            }}>
                                <FileCode2 size={20}/>
                                <strong>{option.label}</strong>
                                <span>{advancedFileHint(option.value)}</span>
                            </button>
                        ))}
                    </div>
                ) : (
                    <>
                        <div className="advanced-toolbar">
                            <select value={props.path} onChange={(event) => props.setPath(event.target.value)}>
                                {props.options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
                            </select>
                            <div className="editor-actions">
                                <span className="language-badge">{languageLabel}</span>
                                <button type="button" className="ghost" onClick={() => editorView && openSearchPanel(editorView)} disabled={!editorView} title="搜索">
                                    <Search size={16}/>搜索
                                </button>
                                <button type="button" className="ghost" onClick={() => editorView && gotoLine(editorView)} disabled={!editorView} title="跳转到行">
                                    <CornerDownRight size={16}/>跳行
                                </button>
                                <button type="button" className="ghost" onClick={() => props.setOpen(false)} disabled={props.busy || props.dirty}>返回文件选择</button>
                                <button className="primary" onClick={props.onSave} disabled={props.busy || !props.dirty}><Save size={16}/>保存</button>
                            </div>
                        </div>
                        <CodeEditor path={props.path} value={props.content} onChange={props.setContent} onReady={setEditorView}/>
                    </>
                )}
            </div>
            {!props.hideFactoryReset && (
                <details className="panel danger-panel">
                    <summary className="danger-summary">
                        <span>
                            <span className="eyebrow">危险操作</span>
                            <strong>恢复出厂设置</strong>
                        </span>
                    </summary>
                    <div className="danger-body">
                        <p className="muted">停止并移除 Hermes 容器，删除 ~/.hermes-dock，然后重新释放内置模板。该操作不可撤销。</p>
                        <label className="reset-confirm">
                            <span>输入「{props.resetConfirmPhrase}」确认</span>
                            <input value={resetConfirmText} onChange={(event) => setResetConfirmText(event.target.value)} disabled={props.busy}/>
                        </label>
                        <button className="danger-button" onClick={factoryReset} disabled={props.busy || !resetConfirmed}><Trash2 size={16}/>恢复出厂设置</button>
                    </div>
                </details>
            )}
        </section>
    );
}

function advancedFileHint(path: string) {
    if (path.endsWith('.env')) return '密钥、平台变量和运行环境';
    if (path.endsWith('SOUL.md')) return '当前助手的人格设定';
    if (path.includes('override')) return '全局 Docker Compose 覆盖';
    return '当前助手的模型、终端和 Hermes 配置';
}
