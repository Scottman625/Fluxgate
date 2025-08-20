/**
 * Queue Widget Component
 * 提供 UI 組件封裝
 */
class QueueWidget {
    constructor(selector, options = {}) {
        this.container = document.querySelector(selector);
        if (!this.container) {
            throw new Error(`Container not found: ${selector}`);
        }

        this.options = {
            apiBase: '/api/v1',
            activityId: 1,
            theme: 'light',
            language: 'zh-TW',
            showProgress: false,
            showETA: false,
            ...options
        };

        // 初始化 SDK
        this.sdk = new QueueSDK({
            apiBase: this.options.apiBase,
            activityId: this.options.activityId,
            config: {
                defaultPollInterval: 2000
            }
        });

        // 渲染 UI
        this.render();
        this.bindEvents();
    }

    render() {
        const themeClass = this.options.theme === 'dark' ? 'queue-widget-dark' : 'queue-widget-light';

        this.container.innerHTML = `
            <div class="queue-widget ${themeClass}">
                <div class="queue-status">
                    <div class="status-indicator" id="status-indicator"></div>
                    <div class="status-text" id="status-text">等待中...</div>
                </div>
                
                ${this.options.showProgress ? `
                <div class="queue-progress">
                    <div class="progress-bar">
                        <div class="progress-fill" id="progress-fill"></div>
                    </div>
                    <div class="progress-text" id="progress-text">0%</div>
                </div>
                ` : ''}
                
                ${this.options.showETA ? `
                <div class="queue-eta" id="queue-eta">
                    <span class="eta-label">預計等待時間:</span>
                    <span class="eta-value" id="eta-value">計算中...</span>
                </div>
                ` : ''}
                
                <div class="queue-position" id="queue-position">
                    <span class="position-label">當前位置:</span>
                    <span class="position-value" id="position-value">-</span>
                </div>
                
                <div class="queue-actions">
                    <button class="btn btn-primary" id="enter-btn">加入排隊</button>
                    <button class="btn btn-secondary" id="refresh-btn">刷新狀態</button>
                </div>
            </div>
        `;

        this.elements = {
            statusIndicator: this.container.querySelector('#status-indicator'),
            statusText: this.container.querySelector('#status-text'),
            progressFill: this.container.querySelector('#progress-fill'),
            progressText: this.container.querySelector('#progress-text'),
            etaValue: this.container.querySelector('#eta-value'),
            positionValue: this.container.querySelector('#position-value'),
            enterBtn: this.container.querySelector('#enter-btn'),
            refreshBtn: this.container.querySelector('#refresh-btn')
        };
    }

    bindEvents() {
        // 綁定 SDK 事件
        this.sdk.on('entered', (data) => {
            this.updateStatus('queuing', '已加入排隊');
            this.updatePosition(data.position);
            this.elements.enterBtn.disabled = true;
            this.elements.enterBtn.textContent = '排隊中...';
        });

        this.sdk.on('statusUpdate', (data) => {
            this.updatePosition(data.position);
            if (data.eta?.estimated_wait_seconds) {
                this.updateETA(data.eta.estimated_wait_seconds);
            }
        });

        this.sdk.on('ready', (data) => {
            this.updateStatus('ready', '可以進入了！');
            this.elements.enterBtn.disabled = false;
            this.elements.enterBtn.textContent = '進入';
            this.elements.enterBtn.className = 'btn btn-success';
        });

        this.sdk.on('error', (error) => {
            this.updateStatus('error', '發生錯誤');
            this.elements.enterBtn.disabled = false;
            this.elements.enterBtn.textContent = '重試';
        });

        // 綁定按鈕事件
        this.elements.enterBtn.addEventListener('click', () => {
            this.enterQueue();
        });

        this.elements.refreshBtn.addEventListener('click', () => {
            this.refresh();
        });
    }

    updateStatus(status, text) {
        this.elements.statusIndicator.className = `status-indicator status-${status}`;
        this.elements.statusText.textContent = text;
    }

    updatePosition(position) {
        if (this.elements.positionValue) {
            this.elements.positionValue.textContent = position || '-';
        }
    }

    updateETA(seconds) {
        if (this.elements.etaValue) {
            const minutes = Math.floor(seconds / 60);
            const remainingSeconds = seconds % 60;
            this.elements.etaValue.textContent = `${minutes}分${remainingSeconds}秒`;
        }
    }

    updateProgress(current, total) {
        if (this.elements.progressFill && this.elements.progressText) {
            const percentage = total > 0 ? Math.round((current / total) * 100) : 0;
            this.elements.progressFill.style.width = `${percentage}%`;
            this.elements.progressText.textContent = `${percentage}%`;
        }
    }

    async enterQueue() {
        try {
            await this.sdk.enterQueue();
        } catch (error) {
            console.error('Failed to enter queue:', error);
        }
    }

    async refresh() {
        try {
            await this.sdk.getQueueStatus();
        } catch (error) {
            console.error('Failed to refresh status:', error);
        }
    }

    destroy() {
        this.sdk.destroy();
        this.container.innerHTML = '';
    }
}
