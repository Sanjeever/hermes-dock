import {useState} from 'react';
import {gotoLine, openSearchPanel} from '@codemirror/search';
import {EditorView} from '@codemirror/view';
import {CornerDownRight, Save, Search} from 'lucide-react';
import {CodeEditor} from '../components/CodeEditor';
import {profileFilePath} from '../utils';

export function SoulPage(props: { profileID: string; content: string; setContent: (value: string) => void; status: string; dirty: boolean; busy: boolean; onSave: () => void; onDiscard: () => void }) {
    const [editorView, setEditorView] = useState<EditorView | null>(null);
    const path = profileFilePath(props.profileID, 'SOUL.md');
    return (
        <section className="advanced-stack">
            <div className="panel">
                <div className="section-head">
                    <div>
                        <p className="eyebrow">人格设定</p>
                        <h2>{path}</h2>
                    </div>
                    <span className={`inline-status ${props.dirty ? 'dirty' : ''}`}>{props.dirty ? '有未保存修改' : props.status}</span>
                </div>
                <div className="advanced-toolbar">
                    <span className="language-badge">Markdown</span>
                    <div className="editor-actions">
                        <button type="button" className="ghost" onClick={() => editorView && openSearchPanel(editorView)} disabled={!editorView} title="搜索">
                            <Search size={16}/>搜索
                        </button>
                        <button type="button" className="ghost" onClick={() => editorView && gotoLine(editorView)} disabled={!editorView} title="跳转到行">
                            <CornerDownRight size={16}/>跳行
                        </button>
                        <button className="ghost" onClick={props.onDiscard} disabled={props.busy || !props.dirty}>放弃修改</button>
                        <button className="primary" onClick={props.onSave} disabled={props.busy || !props.dirty}><Save size={16}/>保存人格</button>
                    </div>
                </div>
                <CodeEditor path={path} value={props.content} onChange={props.setContent} onReady={setEditorView}/>
            </div>
        </section>
    );
}
