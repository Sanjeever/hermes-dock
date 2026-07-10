import {describe, expect, it} from 'vitest';
import {channelStatusKey, closedPolicyValue, disabledPolicyValue, firstBoundPlatform, platformLabel} from './appPolicies';

describe('application policies', () => {
    it('normalizes platform policies', () => {
        expect(closedPolicyValue('open')).toBe('open');
        expect(closedPolicyValue('invalid')).toBe('closed');
        expect(disabledPolicyValue('invalid')).toBe('disabled');
    });

    it('selects the first complete platform binding', () => {
        expect(firstBoundPlatform([
            {key: 'WECOM_BOT_ID', value: 'bot', secret: false},
            {key: 'WECOM_SECRET', value: 'secret', secret: true},
        ])).toBe('wecom');
    });

    it('formats platform and channel identifiers', () => {
        expect(platformLabel('feishu')).toBe('飞书 / Lark');
        expect(channelStatusKey('feishu', 'chat-1', 'test')).toBe('feishu:chat-1:test');
    });
});
