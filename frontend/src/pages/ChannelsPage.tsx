import {useMemo} from 'react';
import {MessageSquarePlus, RefreshCcw} from 'lucide-react';
import type {ChannelFile} from '../types';

export function ChannelsPage({channels, activeProfileName, hasPlatformBinding, weixinHomeChannel, busy, onRefresh, onOpenAssistantPlatforms, onHome, onTest}: {
    channels: ChannelFile;
    activeProfileName: string;
    hasPlatformBinding: boolean;
    weixinHomeChannel: string;
    busy: boolean;
    onRefresh: () => void;
    onOpenAssistantPlatforms: () => void;
    onHome: (platform: string, id: string) => void;
    onTest: (platform: string, id: string) => void;
}) {
    const rows = useMemo(() => Object.entries(channels.platforms || {}).flatMap(([platform, items]) => items.map((item) => ({platform, ...item}))), [channels]);
    return (
        <section className="channel-diagnostics">
            <div className="panel channel-head">
                <div>
                    <p className="eyebrow">通道诊断</p>
                    <h2>{activeProfileName}</h2>
                    <p className="setup-subtitle">通道来自当前助手已绑定的平台，用来确认消息入口是否可用。</p>
                </div>
                <button className="ghost" onClick={onRefresh} disabled={busy}><RefreshCcw size={16}/>刷新通道</button>
            </div>
            {rows.length === 0 ? (
                <div className="panel empty-state">
                    <MessageSquarePlus size={28}/>
                    <h2>{hasPlatformBinding ? '暂未发现通道' : '当前助手还没有绑定平台'}</h2>
                    <p>{hasPlatformBinding ? '启动容器后，从已绑定的平台给助手发送一条消息，再刷新通道。' : '请先回到助手页绑定微信、企业微信或飞书。'}</p>
                    {hasPlatformBinding ? (
                        <button className="primary no-margin" onClick={onRefresh} disabled={busy}><RefreshCcw size={16}/>刷新通道</button>
                    ) : (
                        <button className="primary no-margin" onClick={onOpenAssistantPlatforms} disabled={busy}>去绑定平台</button>
                    )}
                </div>
            ) : (
                <div className="panel">
                    <div className="section-head">
                        <div>
                            <p className="eyebrow">可用会话</p>
                            <h2>{rows.length} 个通道</h2>
                        </div>
                    </div>
                    <div className="table">
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
                </div>
            )}
        </section>
    );
}
