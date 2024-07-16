document.addEventListener('DOMContentLoaded', () => {
    const gameBoard = document.getElementById('game-board');
    const gridSize = 15;
    let socket;
    let playerPosition = { x: 0, y: 0 };
    let playerId = "test";

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

    function setupWebSocket() {
        socket.onopen = function(event) {
            console.log("WebSocket connected");
        };

        socket.onmessage = function(event) {
            console.log("Received Message:", event.data);
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

        // initial player position
        playerPosition = { x: Math.floor(gridSize / 2), y: Math.floor(gridSize / 2) };
        updatePlayerPosition();

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

    function updatePlayerPosition() {
        // remove old position mark to self view
        const oldCell = document.querySelector(`.player[data-player-id="${playerId}"]`);
        if (oldCell) {
            oldCell.classList.remove('player');
            oldCell.removeAttribute('data-player-id');
        }

        // add new position mark to self view
        const newCell = document.getElementById(`cell-${playerPosition.x}-${playerPosition.y}`);
        if (newCell) {
            newCell.classList.add('player');
            newCell.setAttribute('data-player-id', playerId);
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

            if (shouldSend && (newPosition.x !== playerPosition.x || newPosition.y !== playerPosition.y)) {
                const message = JSON.stringify({
                    type: 'playerPosition',
                    content: {
                        id: playerId,
                        position: newPosition
                    }
                });
                socket.send(message);

                // update local position
                playerPosition = newPosition;
                updatePlayerPosition();
            }
        }
    }

    function handleServerUpdate(update) {
        if (update.type === 'gameState') {
            updateGameState(update.content);
        } else if (update.type === 'playerInfo') {
            playerId = update.content.id;
        } else if (update.type === 'playerPosition') {
            handleSelfMoveResponse(update.content);
        }
    }

    function handleSelfMoveResponse(response) {
        console.log(response)
        if (response.Id === playerId) {
            if (response.position.x !== playerPosition.x || response.position.y !== playerPosition.y) {
                // sync correct position from game server
                playerPosition = response.position;
                updatePlayerPosition();
            }
        }
    }

    function updateGameState(gameState) {
        document.querySelectorAll('.player').forEach(el => el.classList.remove('player'));

        for (let player of gameState.players) {
            const cell = document.getElementById(`cell-${player.position.x}-${player.position.y}`);
            if (cell) {
                cell.classList.add('player');
                if (player.id === playerId) {
                    playerPosition = player.position;
                    cell.classList.add('current-player');
                }
            }
        }
    }
});