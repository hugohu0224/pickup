document.addEventListener('DOMContentLoaded', () => {
    const gameBoard = document.getElementById('game-board');
    const gridSize = 15;
    let socket;
    let playerPosition = { x: 0, y: 0 };

    fetch('http://localhost:8080/v1/game/ws-url')
        .then(response => response.json())
        .then(data => {
            // connect to websocket
            const wsUrl = data.url;
            socket = new WebSocket(wsUrl);
            setupWebSocket();

            // init game
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
            console.log("收到消息:", event.data);
        };

        socket.onclose = function(event) {
            console.log("WebSocket closed");
        };

        socket.onerror = function(error) {
            console.error("WebSocket error:", error);
        };
    }

    function initGame() {
        // game board
        for (let y = 0; y < gridSize; y++) {
            for (let x = 0; x < gridSize; x++) {
                const cell = document.createElement('div');
                cell.classList.add('cell');
                cell.id = `cell-${x}-${y}`;
                gameBoard.appendChild(cell);
            }
        }
        updatePlayerPosition();
        document.addEventListener('keydown', handleKeyPress);
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
        movePlayer(direction);
    }

    function movePlayer(direction) {
        const oldCell = document.getElementById(`cell-${playerPosition.x}-${playerPosition.y}`);
        oldCell.classList.remove('player');

        switch(direction) {
            case 'up':
                if (playerPosition.y > 0) playerPosition.y--;
                break;
            case 'down':
                if (playerPosition.y < gridSize - 1) playerPosition.y++;
                break;
            case 'left':
                if (playerPosition.x > 0) playerPosition.x--;
                break;
            case 'right':
                if (playerPosition.x < gridSize - 1) playerPosition.x++;
                break;
        }
        updatePlayerPosition();
        sendPositionToWebSocket(direction);
    }

    function updatePlayerPosition() {
        const newCell = document.getElementById(`cell-${playerPosition.x}-${playerPosition.y}`);
        newCell.classList.add('player');
    }

    function sendPositionToWebSocket() {
        if (socket && socket.readyState === WebSocket.OPEN) {
            const message = JSON.stringify({
                type: 'playerPosition',
                content: {
                    position: playerPosition
                }
            });
            socket.send(message);
        }
    }

});