import {Component, type ErrorInfo, type ReactNode} from 'react';
import {CircleAlert} from 'lucide-react';
import './App.css';

type ErrorBoundaryProps = {
    children: ReactNode;
};

type ErrorBoundaryState = {
    hasError: boolean;
};

class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
    state: ErrorBoundaryState = {hasError: false};

    static getDerivedStateFromError(): ErrorBoundaryState {
        return {hasError: true};
    }

    componentDidCatch(_error: Error, _info: ErrorInfo) {
        console.error('Hermes Dock 界面渲染失败');
    }

    private returnHome = () => {
        window.location.replace(window.location.origin + window.location.pathname);
    };

    render() {
        if (!this.state.hasError) return this.props.children;

        return (
            <main className="app-error-screen" role="alert">
                <section className="app-error-card">
                    <div className="app-error-icon" aria-hidden="true"><CircleAlert size={24}/></div>
                    <div>
                        <p className="eyebrow">Hermes Dock</p>
                        <h1>出错了，请刷新</h1>
                        <p>界面暂时无法继续显示。你的本地配置和数据不会因此被重置。</p>
                    </div>
                    <div className="app-error-actions">
                        <button className="primary no-margin" type="button" onClick={() => window.location.reload()}>重新加载</button>
                        <button className="ghost no-margin" type="button" onClick={this.returnHome}>返回首页</button>
                    </div>
                </section>
            </main>
        );
    }
}

export default ErrorBoundary;
