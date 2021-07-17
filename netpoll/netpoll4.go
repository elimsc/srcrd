package netpoll

// TODO: runtime.poll_runtime_pollSetDeadline
// 设置pd的超时时间，超时时，等待在pd上的g被唤醒并放到当前P或全局队列中
