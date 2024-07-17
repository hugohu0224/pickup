document.addEventListener('DOMContentLoaded', () => {
    const gameBoard = document.getElementById('game-board');
    const gridSize = 15;
    let socket;
    let playerPosition = { x: 0, y: 0 };
    let playerId;
    let players = {};

    fetch('http://localhost:8080/v1/game/ws-url')
        .then(response => response.json())
        .then(data => {
            const wsUrl = data.url;
            socket = new WebSocket(wsUrl);
            setupWebSocket();
            initGame();
        })
        .catch(error => {
            console.error('Error fetching WebSocket URL:', error);
        });

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

    playerId = getUserIdFromCookie();
    if (!playerId) {
        console.error('Player ID not found in cookie');
        return;
    }

    function setupWebSocket() {
        socket.onopen = function(event) {
            console.log("WebSocket connected");
        };

        socket.onmessage = function(event) {
            handleServerUpdate(JSON.parse(event.data));
        };

        socket.onclose = function(event) {
            console.log("WebSocket closed");
        };

        socket.onerror = function(error) {
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

        playerPosition = { x: Math.floor(gridSize / 2), y: Math.floor(gridSize / 2) };
        updatePlayerPosition({ id: playerId, position: playerPosition });

        document.addEventListener('keydown', handleKeyPress);
        sendMoveRequest('initial');
    }

    function handleKeyPress(event) {
        let direction;
        switch(event.key) {
            case 'ArrowUp': direction = 'up'; break;
            case 'ArrowDown': direction = 'down'; break;
            case 'ArrowLeft': direction = 'left'; break;
            case 'ArrowRight': direction = 'right'; break;
            default: return;
        }
        sendMoveRequest(direction);
    }

    function updatePlayerPosition(playerData) {
        if (!playerData || !playerData.position) {
            console.error('Invalid player data:', playerData);
            return;
        }

        const cell = document.getElementById(`cell-${playerData.position.x}-${playerData.position.y}`);
        if (cell) {
            const oldCell = document.querySelector(`.player[data-player-id="${playerData.id}"]`);
            if (oldCell) {
                oldCell.classList.remove('player', 'current-player');
                oldCell.removeAttribute('data-player-id');
            }

            cell.classList.add('player');
            cell.setAttribute('data-player-id', playerData.id);
            if (playerData.id === playerId) {
                playerPosition = playerData.position;
                cell.classList.add('current-player');
            }

            updatePlayerList(playerData);
        }
    }

    function sendMoveRequest(direction) {
        if (socket && socket.readyState === WebSocket.OPEN) {
            let newPosition = { ...playerPosition };
            let shouldSend = false;

            switch(direction) {
                case 'up':
                    if (newPosition.y > 0) {
                        newPosition.y--;
                        shouldSend = true;
                    }
                    break;
                case 'down':
                    if (newPosition.y < gridSize - 1) {
                        newPosition.y++;
                        shouldSend = true;
                    }
                    break;
                case 'left':
                    if (newPosition.x > 0) {
                        newPosition.x--;
                        shouldSend = true;
                    }
                    break;
                case 'right':
                    if (newPosition.x < gridSize - 1) {
                        newPosition.x++;
                        shouldSend = true;
                    }
                    break;
                case 'initial':
                    shouldSend = true;
                    break;
            }

            if (shouldSend) {
                const message = JSON.stringify({
                    type: 'playerPosition',
                    content: {
                        id: playerId,
                        position: newPosition
                    }
                });
                socket.send(message);
            }
        }
    }

    function handleServerUpdate(update) {
        if (update.type === 'gameState') {
            updateGameState(update.content);
        } else if (update.type === 'playerInfo') {
            playerId = update.content.id;
        } else if (update.type === 'playerPosition') {
            updatePlayerPosition(update.content);
        }
    }

    function updateGameState(gameState) {
        document.querySelectorAll('.player').forEach(el => {
            el.classList.remove('player', 'current-player');
            el.removeAttribute('data-player-id');
        });

        for (let player of gameState.players) {
            updatePlayerPosition(player);
        }
    }

    function updatePlayerList(player) {
        const playerList = document.getElementById('player-list');
        if (!playerList) {
            console.error('Player list element not found');
            return;
        }

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
});