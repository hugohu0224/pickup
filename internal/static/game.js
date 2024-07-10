document.addEventListener('DOMContentLoaded', () => {
    const gameBoard = document.getElementById('game-board');
    const gridSize = 15;

    // 創建遊戲板
    for (let y = 0; y < gridSize; y++) {
        for (let x = 0; x < gridSize; x++) {
            const cell = document.createElement('div');
            cell.classList.add('cell');
            cell.id = `cell-${x}-${y}`;
            gameBoard.appendChild(cell);
        }
    }

    // init player position
    let playerPosition = { x: 0, y: 0 };
    updatePlayerPosition();

    document.addEventListener('keydown', (event) => {
        switch(event.key) {
            case 'ArrowUp': movePlayer('up'); break;
            case 'ArrowDown': movePlayer('down'); break;
            case 'ArrowLeft': movePlayer('left'); break;
            case 'ArrowRight': movePlayer('right'); break;
        }
    });

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
    }

    function updatePlayerPosition() {
        const newCell = document.getElementById(`cell-${playerPosition.x}-${playerPosition.y}`);
        newCell.classList.add('player');
    }
});