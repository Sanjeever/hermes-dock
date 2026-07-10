import {describe, expect, it} from 'vitest';
import {createProfileValidationMessage, formatBytes, suggestProfileID} from './assistantUtils';

describe('assistant utilities', () => {
    it('validates profile IDs', () => {
        expect(createProfileValidationMessage('销售', 'sales', false)).toBe('');
        expect(createProfileValidationMessage('销售', 'default', false)).not.toBe('');
        expect(createProfileValidationMessage('销售', 'Sales', false)).not.toBe('');
    });

    it('suggests an unused bounded profile ID', () => {
        expect(suggestProfileID([{id: 'sales'}], 'sales')).toBe('sales-2');
    });

    it('formats skill sizes', () => {
        expect(formatBytes(0)).toBe('0 B');
        expect(formatBytes(1536)).toBe('1.5 KB');
    });
});
