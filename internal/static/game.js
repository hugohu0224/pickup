import {
    sendMoveRequest,
    updateObstacleOnBoard,
    addObstacle,
    handleMoveResponse,
    addItem,
    handleItemCollected,
    notifyUser,
    updatePlayerInList,
} from "./game_action.js";

import {
    updateCountdown,
    handleRoundState,
    handleKeyPress,
    handleWaitingNotification,
    updateTopPlayerOnScoreChange,
} from "./game_round.js"

import {
    shared_state
} from "./game_shared.js";

document.addEventListener('DOMContentLoaded', async () => {
    const messageHandlers = {
        obstaclePosition: addObstacle,
        playerPosition: handleMoveResponse,
        itemPosition: addItem,
        itemCollected: handleItemCollected,
        errorMsg: (content) => notifyUser("Error: " + content.error),
        score: updateSingleScore,
        countdown: updateCountdown,
        roundState: handleRoundState,
        waitingNotification: handleWaitingNotification,
    };

    initializeDOMReferences()

    try {
        const config = await fetchConfig();
        if (!config) throw new Error('Failed to load configuration');

        shared_state.gridSize = config.gridsize || shared_state.gridSize;
        shared_state.playerId = await getUserId();
        if (!shared_state.playerId) throw new Error('Failed to get user ID');

        shared_state.socket = await connectWebSocket(config);
        setupWebSocket();
        initGame();
    } catch (error) {
        console.error('Error initializing game:', error);
        notifyUser("Error: " + error.message);
    }


    function initializeDOMReferences() {
        shared_state.gameBoard = document.getElementById('game-board');
        shared_state.playerList = document.getElementById('player-list');
        shared_state.timerDisplay = document.getElementById('time-left');
    }

    async function fetchConfig(retryCount = 0) {
        const maxRetries = 10;
        const retryDelay = 0.5;
        try {
            const response = await fetch('/v1/config/js');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            return await response.json();
        } catch (error) {
            console.error(`Error fetching config (attempt ${retryCount + 1}):`, error);

            if (retryCount < maxRetries) {
                console.log(`Retrying in ${retryDelay}ms...`);
                await new Promise(resolve => setTimeout(resolve, retryDelay));
                return fetchConfig(retryCount + 1);
            } else {
                console.error('Max retries reached. Failed to fetch config.');
                throw error;
            }
        }
    }

    function connectWebSocket(config, retryCount = 0) {
        return new Promise((resolve, reject) => {
            const maxRetries = 10;
            const retryDelay = 500;
            let retryTimeoutId;

            const socket = new WebSocket(`${config.ws}://${config.endpoint}/v1/game/ws`);

            socket.onopen = () => {
                console.log('WebSocket connected successfully');
                if (retryTimeoutId) {
                    clearTimeout(retryTimeoutId);
                }
                resolve(socket);
            };

            socket.onerror = (error) => {
                console.error('WebSocket connection error:', error);
                if (retryCount < maxRetries) {
                    console.log(`Retrying connection in ${retryDelay}ms... (Attempt ${retryCount + 1})`);
                    retryTimeoutId = setTimeout(() => {
                        connectWebSocket(config, retryCount + 1).then(resolve).catch(reject);
                    }, retryDelay);
                } else {
                    reject(new Error('Max retries reached. WebSocket connection failed.'));
                }
            };
        });
    }

    async function getUserId() {
        try {
            const response = await fetch('/v1/user/id', {
                method: 'GET',
                credentials: 'include',
            });
            if (!response.ok) {
                throw new Error('Failed to fetch user ID');
            }
            const data = await response.json();
            return data.user_id;
        } catch (error) {
            console.error('Error fetching user ID:', error);
            return null;
        }
    }

    function setupWebSocket() {
        shared_state.socket.onopen = () => {
            console.log("WebSocket connected");
        };

        shared_state.socket.onmessage = (event) => {
            const data = JSON.parse(event.data);
            const handler = messageHandlers[data.type];
            if (handler) handler(data.content);
            else console.warn('Unhandled message type:', data.type);
        };

        shared_state.socket.onclose = () => console.log("WebSocket closed");

        shared_state.socket.onerror = (error) => {
            console.error('WebSocket connection error:', error);
        };

    }

    function initGame() {
        if (shared_state.isGameInitialized) return;
        shared_state.isGameInitialized = true;
        createGameBoard();
        shared_state.lastConfirmedPosition = {...shared_state.playerPosition};
        document.addEventListener('keydown', handleKeyPress);
        shared_state.obstacles.forEach(updateObstacleOnBoard);
        sendMoveRequest('initial');
        console.log('Game initialized');
    }

    function createGameBoard() {
        shared_state.gameBoard.innerHTML = Array(shared_state.gridSize).fill().map((_, y) =>
            Array(shared_state.gridSize).fill().map((_, x) =>
                `<div class="cell" id="cell-${x}-${y}"></div>`
            ).join('')
        ).join('');
    }


    function updateSingleScore(scoreUpdate) {
        shared_state.playerScores[scoreUpdate.id] = scoreUpdate.score;
        updatePlayerInList(scoreUpdate.id);
        // for the showWaitingOverlay
        updateTopPlayerOnScoreChange()
    }
});