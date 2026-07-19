import {StrictMode} from 'react';
import {renderToString} from 'react-dom/server';
import App from './App';
import ManualPage from './ManualPage';

export function render(pathname = '/') {
    const Page = pathname === '/manual/' ? ManualPage : App;
    return renderToString(
        <StrictMode>
            <Page />
        </StrictMode>,
    );
}
