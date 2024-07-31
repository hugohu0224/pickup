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

import{
    updateCountdown,
    handleRoundState,
    handleKeyPress,
    handleWaitingNotification,
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
        shared_state.socket = new WebSocket(`ws://${config.endpoint}/v1/game/ws`);
        setupWebSocket();

        shared_state.playerId = getUserIdFromCookie();
        if (!shared_state.playerId) throw new Error('Player ID not found in cookie');
    } catch (error) {
        console.error('Error initializing game:', error);
    }

    function initializeDOMReferences() {
        shared_state.gameBoard = document.getElementById('game-board');
        shared_state.playerList = document.getElementById('player-list');
        shared_state.timerDisplay = document.getElementById('time-left');
    }

    async function fetchConfig() {
        try {
            const response = await fetch('/v1/config/js');
            if (!response.ok) throw new Error('Failed to fetch config');
            return await response.json();
        } catch (error) {
            console.error('Error fetching config:', error);
            return null;
        }
    }

    function getUserIdFromCookie() {
        return document.cookie.split(';')
            .map(cookie => cookie.trim().split('='))
            .find(([name]) => name === 'userId')?.[1] || null;
    }

    function setupWebSocket() {
        shared_state.socket.onopen = () => {
            console.log("WebSocket connected");
            initGame();
        };
        shared_state.socket.onmessage = (event) => {
            const data = JSON.parse(event.data);
            const handler = messageHandlers[data.type];
            if (handler) handler(data.content);
            else console.warn('Unhandled message type:', data.type);
        };
        shared_state.socket.onclose = () => console.log("WebSocket closed");
        shared_state.socket.onerror = (error) => console.error("WebSocket error:", error);
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
    }
});