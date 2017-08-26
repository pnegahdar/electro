import React from 'react';
import ReactDOM from 'react-dom';
import App from './App';
import {unregister} from './registerServiceWorker';

ReactDOM.render(<App />, document.getElementById('root'));
// Disabled for now as subdomains that don't match cache assets.
unregister()
// registerServiceWorker();
