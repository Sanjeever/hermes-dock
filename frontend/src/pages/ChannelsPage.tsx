import {useMemo} from 'react';
import {MessageSquarePlus, RefreshCcw} from 'lucide-react';
import type {ChannelFile} from '../types';

export function ChannelsPage({channels, activeProfileName, hasPlatformBinding, homeChannels, busy, actionStatus, onRefresh, onOpenAssistantPlatforms, onHome, onTest}: {
    channels: ChannelFile;
    activeProfileName: string;
    hasPlatformBinding: boolean;
    homeChannels: Record<string, string>;
    busy: boolean;
    actionStatus: Record<string, string>;
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
                    <p>{hasPlatformBinding ? '启动服务后，从已绑定的平台给助手发送一条消息，再刷新通道。' : '请先回到助手页绑定微信、企业微信、飞书或钉钉。'}</p>
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
                        <div className="table-row table-head">
                            <span>平台</span>
                            <span>通道</span>
                            <span>类型</span>
                            <span>默认通道</span>
                            <span>操作</span>
                        </div>
                        {rows.map((row) => {
                            const homeChannel = homeChannels[row.platform] || '';
                            const supportsHomeChannel = row.platform === 'weixin' || row.platform === 'dingtalk';
                            return <div className="table-row" key={`${row.platform}-${row.id}`}>
                                <code data-label="平台">{row.platform}</code>
                                <span data-label="通道">{row.name || row.id}{row.id === homeChannel && <b className="home-badge">默认</b>}</span>
                                <span data-label="类型">{row.type}</span>
                                {supportsHomeChannel ? (
                                    <button data-label="默认通道" onClick={() => onHome(row.platform, row.id)} disabled={busy || row.id === homeChannel}>{row.id === homeChannel ? '已默认' : '设为默认'}</button>
                                ) : <span className="muted" data-label="默认通道">-</span>}
                                <button data-label="操作" onClick={() => onTest(row.platform, row.id)} disabled={busy}>测试</button>
                                {(actionStatus[channelStatusKey(row.platform, row.id, 'home')] || actionStatus[channelStatusKey(row.platform, row.id, 'test')]) && (
                                    <small className="row-status">{actionStatus[channelStatusKey(row.platform, row.id, 'home')] || actionStatus[channelStatusKey(row.platform, row.id, 'test')]}</small>
                                )}
                            </div>;
                        })}
                    </div>
                </div>
            )}
        </section>
    );
}

function channelStatusKey(platform: string, id: string, action: string) {
    return `${platform}:${id}:${action}`;
}
