import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import WebRoot from './WebRoot'
import ErrorBoundary from './ErrorBoundary'

const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
    <React.StrictMode>
        <ErrorBoundary>
            <WebRoot/>
        </ErrorBoundary>
    </React.StrictMode>
)
