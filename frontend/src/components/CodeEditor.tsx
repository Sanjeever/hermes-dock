import {useEffect, useRef} from 'react';
import {basicSetup} from 'codemirror';
import {yaml} from '@codemirror/lang-yaml';
import {HighlightStyle, StreamLanguage, syntaxHighlighting} from '@codemirror/language';
import {EditorState} from '@codemirror/state';
import {EditorView, keymap} from '@codemirror/view';
import {indentWithTab} from '@codemirror/commands';
import {search} from '@codemirror/search';
import {tags} from '@lezer/highlight';

const editorHighlight = HighlightStyle.define([
    {tag: tags.comment, color: '#75806f', fontStyle: 'italic'},
    {tag: tags.propertyName, color: '#2c6455', fontWeight: '600'},
    {tag: tags.variableName, color: '#2c6455', fontWeight: '600'},
    {tag: tags.string, color: '#7a4d12'},
    {tag: tags.number, color: '#7156a0'},
    {tag: tags.bool, color: '#7156a0'},
    {tag: tags.keyword, color: '#1c6a7a', fontWeight: '600'},
    {tag: tags.operator, color: '#68715f'},
    {tag: tags.punctuation, color: '#68715f'},
]);

const codeEditorTheme = EditorView.theme({
    '&': {
        height: '100%',
        color: '#20251f',
        backgroundColor: '#f7f2e8',
    },
    '&.cm-focused': {
        outline: 'none',
    },
    '.cm-scroller': {
        fontFamily: '"JetBrains Mono", "SF Mono", Menlo, Consolas, monospace',
        fontSize: '13px',
        lineHeight: '1.55',
    },
    '.cm-content': {
        minHeight: '560px',
        padding: '14px 0',
        caretColor: '#20251f',
    },
    '.cm-line': {
        padding: '0 16px',
    },
    '.cm-gutters': {
        backgroundColor: '#eee8da',
        color: '#7a7568',
        borderRight: '0',
    },
    '.cm-lineNumbers .cm-gutterElement': {
        minWidth: '38px',
        padding: '0 12px 0 8px',
    },
    '.cm-activeLine': {
        backgroundColor: '#ece5d6',
    },
    '.cm-activeLineGutter': {
        backgroundColor: '#e3dccd',
        color: '#20251f',
    },
    '.cm-selectionBackground, &.cm-focused .cm-selectionBackground': {
        backgroundColor: '#cfe57f80',
    },
    '.cm-searchMatch': {
        backgroundColor: '#d8f26399',
        outline: '1px solid #98b84b',
    },
    '.cm-searchMatch-selected': {
        backgroundColor: '#c5ee44',
    },
    '.cm-panels': {
        backgroundColor: '#eee8da',
        color: '#20251f',
        borderTop: '0',
        borderBottom: '1px solid #ddd4c4',
    },
    '.cm-panel.cm-search': {
        display: 'flex',
        flexWrap: 'wrap',
        alignItems: 'center',
        gap: '8px',
        padding: '10px',
    },
    '.cm-panel.cm-search input': {
        width: '220px',
        minHeight: '32px',
        backgroundColor: '#fffaf0',
    },
    '.cm-panel.cm-search button': {
        minHeight: '30px',
        padding: '0 10px',
        borderRadius: '6px',
    },
});

const envLanguage = StreamLanguage.define<null>({
    startState: () => null,
    token(stream) {
        if (stream.sol()) {
            stream.eatSpace();
            if (stream.peek() === '#') {
                stream.skipToEnd();
                return 'comment';
            }
            if (stream.match('export')) {
                return 'keyword';
            }
            if (stream.match(/[A-Za-z_][A-Za-z0-9_]*/)) {
                return 'variableName';
            }
        }
        if (stream.peek() === '#') {
            stream.skipToEnd();
            return 'comment';
        }
        if (stream.peek() === '=') {
            stream.next();
            return 'operator';
        }
        if (stream.peek() === '"' || stream.peek() === "'") {
            const quote = stream.next();
            let escaped = false;
            while (!stream.eol()) {
                const next = stream.next();
                if (next === quote && !escaped) break;
                escaped = next === '\\' && !escaped;
                if (next !== '\\') escaped = false;
            }
            return 'string';
        }
        if (stream.match(/[^\s#]+/)) {
            return 'string';
        }
        stream.next();
        return null;
    },
});

export function CodeEditor(props: { path: string; value: string; onChange: (value: string) => void; onReady: (view: EditorView | null) => void }) {
    const hostRef = useRef<HTMLDivElement | null>(null);
    const viewRef = useRef<EditorView | null>(null);
    const onChangeRef = useRef(props.onChange);
    const syncingRef = useRef(false);

    useEffect(() => {
        onChangeRef.current = props.onChange;
    }, [props.onChange]);

    useEffect(() => {
        if (!hostRef.current) return;
        const language = props.path.endsWith('.env') ? envLanguage : yaml();
        const view = new EditorView({
            parent: hostRef.current,
            state: EditorState.create({
                doc: props.value,
                extensions: [
                    basicSetup,
                    keymap.of([indentWithTab]),
                    search({top: true}),
                    language,
                    syntaxHighlighting(editorHighlight),
                    codeEditorTheme,
                    EditorView.lineWrapping,
                    EditorView.updateListener.of((update) => {
                        if (update.docChanged && !syncingRef.current) {
                            onChangeRef.current(update.state.doc.toString());
                        }
                    }),
                ],
            }),
        });
        viewRef.current = view;
        props.onReady(view);
        return () => {
            props.onReady(null);
            view.destroy();
            viewRef.current = null;
        };
    }, [props.path]);

    useEffect(() => {
        const view = viewRef.current;
        if (!view) return;
        const current = view.state.doc.toString();
        if (props.value === current) return;
        syncingRef.current = true;
        view.dispatch({
            changes: {from: 0, to: current.length, insert: props.value},
        });
        syncingRef.current = false;
    }, [props.value]);

    return <div className="code-editor" ref={hostRef}/>;
}
