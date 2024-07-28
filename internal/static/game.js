document.addEventListener('DOMContentLoaded', async () => {
    const gameBoard = document.getElementById('game-board');
    const playerList = document.getElementById('player-list');
    const timerDisplay = document.getElementById('time-left');
    let gridSize = 15;
    let socket;
    let playerPosition = {x: 0, y: 0};
    let lastConfirmedPosition = {x: 0, y: 0};
    let playerId;
    let players = {};
    let playerScores = {};
    let obstacles = [];
    let items = [];
    let isGameInitialized = false;

    const itemHandlers = {
        coin: () => console.log('Coin collected'),
        diamond: () => console.log('Diamond collected')
    };

    const directionMap = {
        'ArrowUp': 'up',
        'ArrowDown': 'down',
        'ArrowLeft': 'left',
        'ArrowRight': 'right'
    };

    const messageHandlers = {
        obstaclePosition: addObstacle,
        playerPosition: handleMoveResponse,
        itemPosition: addItem,
        itemCollected: handleItemCollected,
        gameState: updateGameState,
        errorMsg: (content) => notifyUser("Error: " + content.error),
        score: updateSingleScore,
        countdown: updateCountdown,
        roundState: handleRoundState
    };

    try {
        const config = await fetchConfig();
        if (!config) throw new Error('Failed to load configuration');

        gridSize = config.gridsize || gridSize;
        socket = new WebSocket(`ws://${config.endpoint}/v1/game/ws`);
        setupWebSocket();

        playerId = getUserIdFromCookie();
        if (!playerId) throw new Error('Player ID not found in cookie');
    } catch (error) {
        console.error('Error initializing game:', error);
    }

    function addItem(item) {
        items.push(item);
        updateItemOnBoard(item);
    }

    function updateItemOnBoard(item) {
        const cell = document.getElementById(`cell-${item.position.x}-${item.position.y}`);
        if (cell) {
            cell.classList.add('item', `item-${item.item.type}`);
        }
    }

    function removeItem(item) {
        const index = items.findIndex(i => i.position.x === item.position.x && i.position.y === item.position.y);
        if (index !== -1) {
            const removedItem = items.splice(index, 1)[0];
            const cell = document.getElementById(`cell-${item.position.x}-${item.position.y}`);
            if (cell) {
                cell.classList.remove('item', `item-${removedItem.item.type}`, 'player-on-item');
            }
        }
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

    function addObstacle(obstacle) {
        obstacles.push(obstacle);
        updateObstacleOnBoard(obstacle);
    }

    function updateObstacleOnBoard(obstacle) {
        const cell = document.getElementById(`cell-${obstacle.x}-${obstacle.y}`);
        if (cell) {
            cell.classList.add('obstacle');
        }
    }

    function setupWebSocket() {
        socket.onopen = () => {
            console.log("WebSocket connected");
            initGame();
        };
        socket.onmessage = (event) => {
            const data = JSON.parse(event.data);
            const handler = messageHandlers[data.type];
            if (handler) handler(data.content);
            else console.warn('Unhandled message type:', data.type);
        };
        socket.onclose = () => console.log("WebSocket closed");
        socket.onerror = (error) => console.error("WebSocket error:", error);
    }

    function initGame() {
        if (isGameInitialized) return;
        isGameInitialized = true;

        createGameBoard();
        lastConfirmedPosition = {...playerPosition};
        document.addEventListener('keydown', handleKeyPress);
        obstacles.forEach(updateObstacleOnBoard);
        sendMoveRequest('initial');
        console.log('Game initialized');
    }

    function createGameBoard() {
        gameBoard.innerHTML = Array(gridSize).fill().map((_, y) =>
            Array(gridSize).fill().map((_, x) =>
                `<div class="cell" id="cell-${x}-${y}"></div>`
            ).join('')
        ).join('');
    }

    function sendItemActionRequest() {
        if (socket?.readyState === WebSocket.OPEN) {
            const itemAtPosition = items.find(item =>
                item.position.x === playerPosition.x && item.position.y === playerPosition.y
            );
            if (itemAtPosition) {
                socket.send(JSON.stringify({
                    type: 'itemAction',
                    content: { id: playerId, position: playerPosition }
                }));
            } else {
                console.log("No item at current position to collect");
            }
        }
    }

    function handleKeyPress(event) {
        const direction = directionMap[event.key];
        if (direction) {
            sendMoveRequest(direction);
        } else if (event.key === ' ') {
            sendItemActionRequest();
        }
    }

    function updatePlayerPosition(playerData, status = 'confirmed') {
        if (!playerData?.position) {
            console.error('Invalid player data:', playerData);
            return;
        }

        const oldCell = document.querySelector(`.player[data-player-id="${playerData.id}"]`);
        if (oldCell) {
            oldCell.classList.remove('player', 'current-player', 'other-player', 'unconfirmed');
            oldCell.removeAttribute('data-player-id');
        }

        const cell = document.getElementById(`cell-${playerData.position.x}-${playerData.position.y}`);
        if (!cell) return;

        cell.classList.toggle('player-on-item', cell.classList.contains('item'));

        const existingPlayerId = cell.getAttribute('data-player-id');
        if (existingPlayerId && existingPlayerId !== playerData.id) return;

        cell.classList.add('player');
        cell.setAttribute('data-player-id', playerData.id);

        if (playerData.id === playerId) {
            cell.classList.add('current-player');
            cell.classList.toggle('unconfirmed', status === 'unconfirmed');
        } else {
            cell.classList.add('other-player');
        }

        cell.classList.toggle('player-on-coin', cell.classList.contains('coin'));

        players[playerData.id] = playerData.position;
        updatePlayerInList(playerData.id);
    }

    function sendMoveRequest(direction) {
        if (socket?.readyState === WebSocket.OPEN) {
            const newPosition = direction === 'initial' ? playerPosition : calculateNewPosition(playerPosition, direction);
            if (isValidMove(playerPosition, newPosition)) {
                updatePlayerPosition({id: playerId, position: newPosition}, 'unconfirmed');
                socket.send(JSON.stringify({
                    type: 'playerPosition',
                    content: { id: playerId, position: newPosition }
                }));
            }
        } else {
            console.log('WebSocket is not open. Cannot send move request.');
        }
    }

    function calculateNewPosition(currentPosition, direction) {
        const newPosition = {...currentPosition};
        switch (direction) {
            case 'up': newPosition.y = Math.max(0, newPosition.y - 1); break;
            case 'down': newPosition.y = Math.min(gridSize - 1, newPosition.y + 1); break;
            case 'left': newPosition.x = Math.max(0, newPosition.x - 1); break;
            case 'right': newPosition.x = Math.min(gridSize - 1, newPosition.x + 1); break;
        }
        return newPosition;
    }

    function isValidMove(currentPosition, newPosition) {
        return newPosition.x >= 0 && newPosition.x < gridSize &&
            newPosition.y >= 0 && newPosition.y < gridSize &&
            (Math.abs(newPosition.x - currentPosition.x) + Math.abs(newPosition.y - currentPosition.y) === 1);
    }

    function handleMoveResponse(response) {
        if (response.id === playerId) {
            if (response.valid) {
                lastConfirmedPosition = response.position;
                playerPosition = response.position;
            } else {
                playerPosition = lastConfirmedPosition;
                notifyUser("Invalid move: " + (response.reason || "Unknown reason"));
            }
            updatePlayerPosition({id: playerId, position: playerPosition});
        } else {
            updatePlayerPosition(response);
        }
    }

    function updateGameState(gameState) {
        gameState.players.forEach(updatePlayerPosition);
        if (gameState.currentPlayer?.id === playerId) {
            playerPosition = gameState.currentPlayer.position;
            lastConfirmedPosition = {...playerPosition};
        }
    }

    function notifyUser(message) {
        console.log(message);
    }

    function updatePlayerInList(userId) {
        let playerElement = document.getElementById(`player-${userId}`);
        if (!playerElement) {
            playerElement = document.createElement('div');
            playerElement.id = `player-${userId}`;
            playerElement.className = 'player-item';
            playerList.appendChild(playerElement);
        }
        const score = playerScores[userId] || 0;
        const isCurrentPlayer = userId === playerId;

        playerElement.className = `player-item${isCurrentPlayer ? ' current-player' : ''}`;
        playerElement.textContent = `Player ${userId}: Score ${score}${isCurrentPlayer ? ' (You)' : ''}`;
    }

    function handleItemCollected(data) {
        if (data.valid) {
            removeItem(data);
            const handler = itemHandlers[data.item.type];
            if (handler) handler();
        } else {
            notifyUser("Failed to collect item: " + data.reason);
        }
    }

    function updateSingleScore(scoreUpdate) {
        playerScores[scoreUpdate.id] = scoreUpdate.score;
        updatePlayerInList(scoreUpdate.id);
    }

    function updateCountdown(countdownUpdate) {
        const remainingTime = countdownUpdate.remainingTime;
        const currentState = countdownUpdate.currentState;

        const timerDisplay = document.getElementById('time-left');
        timerDisplay.textContent = remainingTime;

        const gameInfo = document.getElementById('game-info');
        const stateDisplay = gameInfo.querySelector('.game-state');
        if (!stateDisplay) {
            const stateElement = document.createElement('div');
            stateElement.className = 'game-state';
            gameInfo.appendChild(stateElement);
        }
        stateDisplay.textContent = currentState === 'waiting' ? 'Waiting for next round' : 'Game in progress';

        const gameMessages = document.getElementById('game-messages');
        const message = document.createElement('p');
        message.textContent = `${currentState === 'waiting' ? 'Waiting' : 'Playing'} - ${remainingTime}s left`;
        gameMessages.appendChild(message);

        while (gameMessages.children.length > 5) {
            gameMessages.removeChild(gameMessages.firstChild);
        }

        if (remainingTime <= 0) {
            if (currentState === 'waiting') {
                console.log('New round is about to start!');

            } else {
                console.log('Current round is ending!');
            }
        }
    }

    function handleRoundState(roundState) {
        const state = roundState.state;
        const currentTime = new Date(roundState.currentTime);
        const endTime = new Date(roundState.endTime);

        console.log(`Round state changed to: ${state}`);

        const gameInfo = document.getElementById('game-info');
        const stateDisplay = gameInfo.querySelector('.game-state');
        if (!stateDisplay) {
            const stateElement = document.createElement('div');
            stateElement.className = 'game-state';
            gameInfo.appendChild(stateElement);
        }
        stateDisplay.textContent = `Game ${state}`;

        const gameMessages = document.getElementById('game-messages');
        const message = document.createElement('p');
        message.textContent = `Round ${state} - Ends at ${endTime.toLocaleTimeString()}`;
        gameMessages.appendChild(message);

        while (gameMessages.children.length > 5) {
            gameMessages.removeChild(gameMessages.firstChild);
        }

        if (state === 'waiting') {
            resetGameData();
        } else if (state === 'playing') {
            resetGameData();
            startNewRound();
        } else if (state === 'ended') {
            endCurrentRound();
        }
    }

    function handleRoundState(roundState) {
        const state = roundState.state;
        const currentTime = new Date(roundState.currentTime);
        const endTime = new Date(roundState.endTime);

        console.log(`Round state changed to: ${state}`);

        const gameInfo = document.getElementById('game-info');
        const stateDisplay = gameInfo.querySelector('.game-state');
        if (!stateDisplay) {
            const stateElement = document.createElement('div');
            stateElement.className = 'game-state';
            gameInfo.appendChild(stateElement);
        }
        stateDisplay.textContent = `Game ${state}`;

        const gameMessages = document.getElementById('game-messages');
        const message = document.createElement('p');
        message.textContent = `Round ${state} - Ends at ${endTime.toLocaleTimeString()}`;
        gameMessages.appendChild(message);

        while (gameMessages.children.length > 5) {
            gameMessages.removeChild(gameMessages.firstChild);
        }

        if (state === 'waiting') {
            resetGameData();
        } else if (state === 'playing') {
        } else if (state === 'ended') {
        }
    }

    function resetGameData() {
        playerPosition = {x: 0, y: 0};
        lastConfirmedPosition = {x: 0, y: 0};
        players = {};
        playerScores = {};
        obstacles = [];
        items = [];


        const gameBoard = document.getElementById('game-board');
        const cells = gameBoard.getElementsByClassName('cell');
        Array.from(cells).forEach(cell => {
            cell.classList.remove('player', 'current-player', 'other-player', 'obstacle', 'item', 'item-coin', 'item-diamond', 'player-on-item');
            cell.removeAttribute('data-player-id');
        });

        const playerList = document.getElementById('player-list');
        playerList.innerHTML = '';


        const gameMessages = document.getElementById('game-messages');
        gameMessages.innerHTML = '';

        const timerDisplay = document.getElementById('time-left');
        if (timerDisplay) {
            timerDisplay.textContent = '';
        }

        const gameInfo = document.getElementById('game-info');
        const stateDisplay = gameInfo.querySelector('.game-state');
        if (stateDisplay) {
            stateDisplay.textContent = '';
        }

        const modal = document.getElementById('game-modal');
        if (modal) {
            modal.style.display = 'none';
        }

        document.removeEventListener('keydown', handleKeyPress);

        console.log('Game data and DOM elements reset');
    }
});