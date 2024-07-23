document.addEventListener('DOMContentLoaded', async () => {
    const gameBoard = document.getElementById('game-board');
    let gridSize = 15;
    let socket;
    let playerPosition = {x: 0, y: 0};
    let lastConfirmedPosition = {x: 0, y: 0};
    let playerId;
    let players = {};
    let obstacles = [];
    let items = [];

    function addItem(item) {
        items.push(item);
        updateItemOnBoard(item);
    }

    function updateItemOnBoard(item) {
        const cellId = `cell-${item.position.x}-${item.position.y}`;
        const cell = document.getElementById(cellId);
        if (cell) {
            cell.classList.add('item', `item-${item.item.type}`);
        }
    }

    function removeItem(item) {
        const index = items.findIndex(i => i.position.x === item.position.x && i.position.y === item.position.y);
        if (index !== -1) {
            const removedItem = items.splice(index, 1)[0];
            const cellId = `cell-${item.position.x}-${item.position.y}`;
            const cell = document.getElementById(cellId);
            if (cell) {
                cell.classList.remove('item', `item-${removedItem.item.type}`, 'player-on-item');
            }
        }
    }

    async function fetchConfig() {
        try {
            const response = await fetch('/v1/config/js');
            if (!response.ok) {
                throw new Error('Failed to fetch config');
            }
            return await response.json();
        } catch (error) {
            console.error('Error fetching config:', error);
            return null;
        }
    }

    try {
        const config = await fetchConfig();
        if (!config) {
            throw new Error('Failed to load configuration');
        }

        gridSize = config.gridsize || gridSize;
        const wsUrl = `ws://${config.endpoint}/v1/game/ws`;
        socket = new WebSocket(wsUrl);
        setupWebSocket();

    } catch (error) {
        console.error('Error initializing game:', error);
        return;
    }

    function getUserIdFromCookie() {
        const cookies = document.cookie.split(';');
        for (let cookie of cookies) {
            const [name, value] = cookie.trim().split('=');
            if (name === 'userId') {
                return value;
            }
        }
        return null;
    }

    // assign userId to playerId
    playerId = getUserIdFromCookie();
    if (!playerId) {
        console.error('Player ID not found in cookie');
        return;
    }

    function addObstacle(obstacle) {
        obstacles.push(obstacle);
        updateObstacleOnBoard(obstacle);
    }

    function updateObstacleOnBoard(obstacle) {
        const cellId = `cell-${obstacle.x}-${obstacle.y}`;
        const cell = document.getElementById(cellId);
        if (cell) {
            cell.classList.add('obstacle');
        }
    }

    function setupWebSocket() {
        socket.onopen = function (event) {
            console.log("WebSocket connected");
            initGame();
        };

        socket.onmessage = function(event) {
            const data = JSON.parse(event.data);
            switch(data.type) {
                case 'obstaclePosition':
                    addObstacle(data.content);
                    break;
                case 'playerPosition':
                    handleMoveResponse(data.content);
                    break;
                case 'itemPosition':
                    addItem(data.content);
                    break;
                case 'itemCollected':
                    handleItemCollected(data.content);
                    break;
                default:
                    handleServerUpdate(data);
            }
        };

        socket.onclose = function (event) {
            console.log("WebSocket closed");
        };

        socket.onerror = function (error) {
            console.error("WebSocket error:", error);
        };
    }

    function initGame() {
        for (let y = 0; y < gridSize; y++) {
            for (let x = 0; x < gridSize; x++) {
                const cell = document.createElement('div');
                cell.classList.add('cell');
                cell.id = `cell-${x}-${y}`;
                gameBoard.appendChild(cell);
            }
        }

        // init position
        lastConfirmedPosition = {...playerPosition};

        document.addEventListener('keydown', handleKeyPress);

        obstacles.forEach(updateObstacleOnBoard);

        if (socket.readyState === WebSocket.OPEN) {
            sendMoveRequest('initial');
        } else {
            console.log('WebSocket not ready, waiting...');
            socket.addEventListener('open', () => {
                sendMoveRequest('initial');
            });
        }
        console.log('Game initialized');
    }

    function sendItemActionRequest() {
        if (socket && socket.readyState === WebSocket.OPEN) {
            const itemAtPosition = items.find(item => item.position.x === playerPosition.x && item.position.y === playerPosition.y);
            if (itemAtPosition) {
                const message = JSON.stringify({
                    type: 'itemAction',
                    content: {
                        id: playerId,
                        position: playerPosition
                    }
                });
                socket.send(message);
            } else {
                console.log("No item at current position to collect");
            }
        }
    }

    function handleKeyPress(event) {
        let direction;
        switch (event.key) {
            case 'ArrowUp':
                direction = 'up';
                break;
            case 'ArrowDown':
                direction = 'down';
                break;
            case 'ArrowLeft':
                direction = 'left';
                break;
            case 'ArrowRight':
                direction = 'right';
                break;
            case ' ':
                sendItemActionRequest();
                return;
            default:
                return;
        }
        if (direction) {
            sendMoveRequest(direction);
        }
    }

    function updatePlayerPosition(playerData, status = 'confirmed') {
        if (!playerData || !playerData.position) {
            console.error('Invalid player data:', playerData);
            return;
        }



        // remove old position
        const oldCell = document.querySelector(`.player[data-player-id="${playerData.id}"]`);
        if (oldCell) {
            oldCell.classList.remove('player', 'current-player', 'other-player', 'unconfirmed');
            oldCell.removeAttribute('data-player-id');
        }

        // update position
        const cellId = `cell-${playerData.position.x}-${playerData.position.y}`;
        const cell = document.getElementById(cellId);

        if (cell.classList.contains('item')) {
            cell.classList.add('player-on-item');
        } else {
            cell.classList.remove('player-on-item');
        }

        if (cell) {
            // check if occupied
            const existingPlayerId = cell.getAttribute('data-player-id');
            if (existingPlayerId && existingPlayerId !== playerData.id) {
                return;
            }

            cell.classList.add('player');
            cell.setAttribute('data-player-id', playerData.id);

            if (playerData.id === playerId) {
                cell.classList.add('current-player');
                cell.classList.remove('other-player');
                if (status === 'unconfirmed') {
                    cell.classList.add('unconfirmed');
                } else {
                    cell.classList.remove('unconfirmed');
                }
            }

            else {
                cell.classList.add('other-player');
                cell.classList.remove('current-player', 'unconfirmed');
            }

            // keep show coin
            if (cell.classList.contains('coin')) {
                cell.classList.add('player-on-coin');
            }
        }

        players[playerData.id] = playerData.position;
        updatePlayerList(playerData);
    }

    function sendMoveRequest(direction) {
        if (socket && socket.readyState === WebSocket.OPEN) {
            let newPosition;
            if (direction === 'initial') {
                newPosition = playerPosition;
            } else {
                newPosition = calculateNewPosition(playerPosition, direction);
            }

            if (isValidMove(playerPosition, newPosition)) {
                updatePlayerPosition({id: playerId, position: newPosition}, 'unconfirmed');

                const message = JSON.stringify({
                    type: 'playerPosition',
                    content: {
                        id: playerId,
                        position: newPosition
                    }
                });
                socket.send(message);
            }
        } else {
            console.log('WebSocket is not open. Cannot send move request.');
        }
    }

    function calculateNewPosition(currentPosition, direction) {
        let newPosition = {...currentPosition};
        switch (direction) {
            case 'up':
                newPosition.y = Math.max(0, newPosition.y - 1);
                break;
            case 'down':
                newPosition.y = Math.min(gridSize - 1, newPosition.y + 1);
                break;
            case 'left':
                newPosition.x = Math.max(0, newPosition.x - 1);
                break;
            case 'right':
                newPosition.x = Math.min(gridSize - 1, newPosition.x + 1);
                break;
        }
        return newPosition;
    }

    function isValidMove(currentPosition, newPosition) {
        return (
            newPosition.x >= 0 && newPosition.x < gridSize &&
            newPosition.y >= 0 && newPosition.y < gridSize &&
            (Math.abs(newPosition.x - currentPosition.x) + Math.abs(newPosition.y - currentPosition.y) === 1)
        );
    }

    function handleServerUpdate(update) {
        if (update.type === 'gameState') {
            updateGameState(update.content);
        } else if (update.type === 'playerInfo') {
            playerId = update.content.id;
        } else if (update.type === 'playerPosition') {
            handleMoveResponse(update.content);
        } else if (update.type === 'error') {
        }
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
        for (let player of gameState.players) {
            updatePlayerPosition(player);
        }

        if (gameState.currentPlayer && gameState.currentPlayer.id === playerId) {
            playerPosition = gameState.currentPlayer.position;
            lastConfirmedPosition = {...playerPosition};
        }
    }

    function notifyUser(message) {
        console.log(message);
    }

    function updatePlayerList(player) {
        const playerList = document.getElementById('player-list');
        let playerElement = document.getElementById(`player-${player.id}`);

        if (!playerElement) {
            playerElement = document.createElement('div');
            playerElement.id = `player-${player.id}`;
            playerList.appendChild(playerElement);
        }

        playerElement.textContent = `Player ${player.id}: (${player.position.x}, ${player.position.y})`;
        if (player.id === playerId) {
            playerElement.textContent += ' (You)';
        }
    }

    function handleItemCollected(data) {
        if (data.valid) {
            removeItem(data);

            console.log(JSON.stringify(data));

            switch (data.item.type) {
                case 'coin':
                    break
                case 'diamond':
                    break
            }
        } else {
            notifyUser("Failed to collect coin: " + data.reason);
        }
    }

});