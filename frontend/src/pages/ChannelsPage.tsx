import {useMemo} from 'react';
import {RefreshCcw} from 'lucide-react';
import type {ChannelFile} from '../types';

export function ChannelsPage({channels, weixinHomeChannel, busy, onRefresh, onHome, onTest}: {
    channels: ChannelFile;
    weixinHomeChannel: string;
    busy: boolean;
    onRefresh: () => void;
    onHome: (platform: string, id: string) => void;
    onTest: (platform: string, id: string) => void;
}) {
    const rows = useMemo(() => Object.entries(channels.platforms || {}).flatMap(([platform, items]) => items.map((item) => ({platform, ...item}))), [channels]);
    return (
        <section className="panel">
            <div className="section-head">
                <div>
                    <p className="eyebrow">通道目录</p>
                    <h2>可用会话</h2>
                </div>
                <button className="ghost" onClick={onRefresh} disabled={busy}><RefreshCcw size={16}/>刷新</button>
            </div>
            <div className="table">
                {rows.length === 0 && <p className="muted">还没有发现通道。请先启动 Hermes，并从微信或企业微信发送一条消息。</p>}
                {rows.map((row) => (
                    <div className="table-row" key={`${row.platform}-${row.id}`}>
                        <code>{row.platform}</code>
                        <span>{row.name || row.id}{row.platform === 'weixin' && row.id === weixinHomeChannel && <b className="home-badge">默认</b>}</span>
                        <span>{row.type}</span>
                        {row.platform === 'weixin' ? (
                            <button onClick={() => onHome(row.platform, row.id)} disabled={busy || row.id === weixinHomeChannel}>{row.id === weixinHomeChannel ? '已默认' : '设为默认'}</button>
                        ) : <span className="muted">-</span>}
                        <button onClick={() => onTest(row.platform, row.id)} disabled={busy}>测试</button>
                    </div>
                ))}
            </div>
        </section>
    );
}
