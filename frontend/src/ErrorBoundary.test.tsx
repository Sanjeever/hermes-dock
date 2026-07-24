import {render, screen} from '@testing-library/react';
import {afterEach, describe, expect, it, vi} from 'vitest';
import ErrorBoundary from './ErrorBoundary';

function BrokenView(): JSX.Element {
    throw new Error('render failed');
}

describe('ErrorBoundary', () => {
	afterEach(() => { vi.restoreAllMocks(); });

    it('shows a recoverable Chinese fallback when rendering fails', () => {
		vi.spyOn(console, 'error').mockImplementation(() => undefined);
        render(<ErrorBoundary><BrokenView/></ErrorBoundary>);

        expect(screen.getByText('出错了，请刷新')).toBeTruthy();
        expect(screen.getByRole('button', {name: '重新加载'})).toBeTruthy();
        expect(screen.getByRole('button', {name: '返回首页'})).toBeTruthy();
    });
});
