/**
 * Queue System JavaScript SDK
 * 提供排隊功能的前端整合
 */
class QueueSDK {
    constructor(options = {}) {
        this.apiBase = options.apiBase || '/api/v1';
        this.activityId = options.activityId;
        this.userHash = options.userHash || this.generateUserHash();
        this.fingerprint = options.fingerprint || this.generateFingerprint();
        
        // 狀態管理
        this.status = 'idle'; // idle, queuing, ready, error
        this.queueData = null;
        this.pollTimer = null;
        this.listeners = {};
        
        // 配置
        this.config = {
            maxRetries: 3,
            retryDelay: 1000,
            defaultPollInterval: 2000,
            ...options.config
        };

        // 自動重連機制
        this.retryCount = 0;
        this.isDestroyed = false;
    }

    /**
     * 進入隊列
     */
    async enterQueue() {
        if (this.status === 'queuing') {
            console.warn('Already in queue');
            return this.queueData;
        }

        try {
            this.setStatus('queuing');
            this.retryCount = 0;

            const response = await this.makeRequest('POST', '/queue/enter', {
                activity_id: this.activityId,
                user_hash: this.userHash,
                fingerprint: this.fingerprint
            });

            if (response.success) {
                this.queueData = response.data;
                this.emit('entered', this.queueData);
                
                // 開始輪詢狀態
                this.startPolling();
                return this.queueData;
            } else {
                throw new Error(response.message || 'Failed to enter queue');
            }
        } catch (error) {
            this.setStatus('error');
            this.emit('error', error);
            throw error;
        }
    }

    /**
     * 開始輪詢隊列狀態
     */
    startPolling() {
        if (!this.queueData || this.isDestroyed) return;

        this.stopPolling(); // 確保沒有重複的輪詢

        const poll = async () => {
            try {
                const status = await this.getQueueStatus();
                this.handleStatusUpdate(status);
            } catch (error) {
                this.handlePollError(error);
            }
        };

        // 立即執行一次
        poll();
    }

    /**
     * 停止輪詢
     */
    stopPolling() {
        if (this.pollTimer) {
            clearTimeout(this.pollTimer);
            this.pollTimer = null;
        }
    }

    /**
     * 獲取隊列狀態
     */
    async getQueueStatus() {
        if (!this.queueData) {
            throw new Error('Not in queue');
        }

        const params = new URLSearchParams({
            activity_id: this.activityId,
            seq: this.queueData.seq,
            session_id: this.queueData.session_id
        });

        const response = await this.makeRequest('GET', `/queue/status?${params}`);
        
        if (response.success) {
            return response.data;
        } else {
            throw new Error(response.message || 'Failed to get queue status');
        }
    }

    /**
     * 處理狀態更新
     */
    handleStatusUpdate(status) {
        const prevPosition = this.queueData?.position || 0;
        
        // 更新隊列數據
        this.queueData = { ...this.queueData, ...status };
        
        // 發送狀態更新事件
        this.emit('statusUpdate', {
            ...status,
            prevPosition,
            positionChanged: status.position !== prevPosition
        });

        // 檢查是否可以進入
        if (status.can_enter) {
            this.setStatus('ready');
            this.emit('ready', status);
            this.stopPolling();
            return;
        }

        // 計算下次輪詢間隔
        const pollInterval = status.eta?.next_poll_interval_ms || this.config.defaultPollInterval;
        
        // 設置下次輪詢
        this.pollTimer = setTimeout(() => {
            if (!this.isDestroyed) {
                this.startPolling();
            }
        }, pollInterval);

        // 重置重試計數
        this.retryCount = 0;
    }

    /**
     * 處理輪詢錯誤
     */
    handlePollError(error) {
        console.error('Poll error:', error);
        
        this.retryCount++;
        
        if (this.retryCount >= this.config.maxRetries) {
            this.setStatus('error');
            this.emit('error', error);
            return;
        }

        // 指數退避重試
        const delay = this.config.retryDelay * Math.pow(2, this.retryCount - 1);
        
        this.pollTimer = setTimeout(() => {
            if (!this.isDestroyed) {
                this.startPolling();
            }
        }, delay);
    }

    /**
     * 設置狀態
     */
    setStatus(newStatus) {
        const oldStatus = this.status;
        this.status = newStatus;
        this.emit('statusChanged', { oldStatus, newStatus });
    }

    /**
     * 事件監聽
     */
    on(event, callback) {
        if (!this.listeners[event]) {
            this.listeners[event] = [];
        }
        this.listeners[event].push(callback);
    }

    /**
     * 移除事件監聽
     */
    off(event, callback) {
        if (!this.listeners[event]) return;
        
        const index = this.listeners[event].indexOf(callback);
        if (index > -1) {
            this.listeners[event].splice(index, 1);
        }
    }

    /**
     * 發送事件
     */
    emit(event, data) {
        if (!this.listeners[event]) return;
        
        this.listeners[event].forEach(callback => {
            try {
                callback(data);
            } catch (error) {
                console.error(`Error in event listener for ${event}:`, error);
            }
        });
    }

    /**
     * 發送 HTTP 請求
     */
    async makeRequest(method, path, data = null) {
        const url = `${this.apiBase}${path}`;
        const options = {
            method,
            headers: {
                'Content-Type': 'application/json',
                'X-Request-ID': this.generateRequestId()
            }
        };

        if (data && (method === 'POST' || method === 'PUT')) {
            options.body = JSON.stringify(data);
        }

        const response = await fetch(url, options);
        const result = await response.json();

        if (!response.ok) {
            throw new Error(result.message || `HTTP ${response.status}`);
        }

        return result;
    }

    /**
     * 生成用戶雜湊
     */
    generateUserHash() {
        // 基於瀏覽器特徵生成穩定的用戶標識
        const features = [
            navigator.userAgent,
            navigator.language,
            screen.width + 'x' + screen.height,
            new Date().getTimezoneOffset(),
            navigator.platform
        ].join('|');

        return this.simpleHash(features);
    }

    /**
     * 生成設備指紋
     */
    generateFingerprint() {
        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');
        ctx.textBaseline = 'top';
        ctx.font = '14px Arial';
        ctx.fillText('Queue fingerprint', 2, 2);
        
        const fingerprint = [
            canvas.toDataURL(),
            navigator.hardwareConcurrency || 0,
            navigator.deviceMemory || 0,
            window.devicePixelRatio || 1
        ].join('|');

        return this.simpleHash(fingerprint);
    }

    /**
     * 簡單雜湊函數
     */
    simpleHash(str) {
        let hash = 0;
        for (let i = 0; i < str.length; i++) {
            const char = str.charCodeAt(i);
            hash = ((hash << 5) - hash) + char;
            hash = hash & hash; // 轉換為 32 位整數
        }
        return Math.abs(hash).toString(36);
    }

    /**
     * 生成請求 ID
     */
    generateRequestId() {
        return 'sdk-' + Date.now() + '-' + Math.random().toString(36).substr(2, 9);
    }

    /**
     * 獲取當前狀態
     */
    getStatus() {
        return {
            status: this.status,
            queueData: this.queueData,
            activityId: this.activityId,
            userHash: this.userHash
        };
    }

    /**
     * 銷毀實例
     */
    destroy() {
        this.isDestroyed = true;
        this.stopPolling();
        this.listeners = {};
        this.queueData = null;
        this.setStatus('idle');
    }
}

// 導出為全域變數或模組
if (typeof module !== 'undefined' && module.exports) {
    module.exports = QueueSDK;
} else if (typeof window !== 'undefined') {
    window.QueueSDK = QueueSDK;
}
