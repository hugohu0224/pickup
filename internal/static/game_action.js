import { shared_state } from "./game_shared.js";

export function updatePlayerPosition(playerData, status = 'confirmed') {
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

    if (playerData.id === shared_state.playerId) {
        cell.classList.add('current-player');
        cell.classList.toggle('unconfirmed', status === 'unconfirmed');
    } else {
        cell.classList.add('other-player');
    }

    cell.classList.toggle('player-on-coin', cell.classList.contains('coin'));

    shared_state.players[playerData.id] = playerData.position;
    updatePlayerInList(playerData.id);
}

export function sendMoveRequest(direction) {
    if (shared_state.socket?.readyState === WebSocket.OPEN) {
        const newPosition = direction === 'initial' ? shared_state.playerPosition : calculateNewPosition(shared_state.playerPosition, direction);
        if (isValidMove(shared_state.playerPosition, newPosition)) {
            updatePlayerPosition({id: shared_state.playerId, position: newPosition}, 'unconfirmed');
            shared_state.socket.send(JSON.stringify({
                type: 'playerPosition',
                content: {id: shared_state.playerId, position: newPosition}
            }));
        }
    } else {
        console.log('WebSocket is not open. Cannot send move request.');
    }
}



export function calculateNewPosition(currentPosition, direction) {
    const newPosition = {...currentPosition};
    switch (direction) {
        case 'up':
            newPosition.y = Math.max(0, newPosition.y - 1);
            break;
        case 'down':
            newPosition.y = Math.min(shared_state.gridSize - 1, newPosition.y + 1);
            break;
        case 'left':
            newPosition.x = Math.max(0, newPosition.x - 1);
            break;
        case 'right':
            newPosition.x = Math.min(shared_state.gridSize - 1, newPosition.x + 1);
            break;
    }
    return newPosition;
}

export function isValidMove(currentPosition, newPosition) {
    return newPosition.x >= 0 && newPosition.x < shared_state.gridSize &&
        newPosition.y >= 0 && newPosition.y < shared_state.gridSize &&
        (Math.abs(newPosition.x - currentPosition.x) + Math.abs(newPosition.y - currentPosition.y) === 1);
}

export function updatePlayerInList(userId) {
    let playerElement = document.getElementById(`player-${userId}`);
    if (!playerElement) {
        playerElement = document.createElement('div');
        playerElement.id = `player-${userId}`;
        playerElement.className = 'player-item';
        shared_state.playerList.appendChild(playerElement);
    }
    const score = shared_state.playerScores[userId] || 0;
    const isCurrentPlayer = userId === shared_state.playerId;

    playerElement.className = `player-item${isCurrentPlayer ? ' current-player' : ''}`;
    playerElement.textContent = `Player ${userId}: Score ${score}${isCurrentPlayer ? ' (You)' : ''}`;

}

export function sendItemActionRequest() {
    if (shared_state.socket?.readyState === WebSocket.OPEN) {
        const itemAtPosition = shared_state.items.find(item =>
            item.position.x === shared_state.playerPosition.x && item.position.y === shared_state.playerPosition.y
        );
        if (itemAtPosition) {
            shared_state.socket.send(JSON.stringify({
                type: 'itemAction',
                content: {id: shared_state.playerId, position: shared_state.playerPosition}
            }));
        } else {
            console.log("No item at current position to collect");
        }
    }
}

export function notifyUser(message) {
    console.log(message);
}

export function alertUser(message) {
    alert(message);
}

export function handleMoveResponse(response) {
    if (response.id === shared_state.playerId) {
        if (response.valid) {
            shared_state.lastConfirmedPosition = response.position;
            shared_state.playerPosition = response.position;
        } else {
            shared_state.playerPosition = shared_state.lastConfirmedPosition;
            notifyUser("Invalid move: " + (response.reason || "Unknown reason"));
        }
        updatePlayerPosition({id: shared_state.playerId, position: shared_state.playerPosition});
    } else {
        updatePlayerPosition(response);
    }
}

const itemHandlers = {
    coin: () => console.log('Coin collected'),
    diamond: () => console.log('Diamond collected')
};

export function handleItemCollected(data) {
    if (data.valid) {
        removeItem(data);
        const handler = itemHandlers[data.item.type];
        if (handler) handler();
    } else {
        notifyUser("Failed to collect item: " + data.reason);
    }
}

export function removeItem(item) {
    const index = shared_state.items.findIndex(i => i.position.x === item.position.x && i.position.y === item.position.y);
    if (index !== -1) {
        const removedItem = shared_state.items.splice(index, 1)[0];
        const cell = document.getElementById(`cell-${item.position.x}-${item.position.y}`);
        if (cell) {
            cell.classList.remove('item', `item-${removedItem.item.type}`, 'player-on-item');
        }
    }
}

export function addObstacle(obstacle) {
    shared_state.obstacles.push(obstacle);
    updateObstacleOnBoard(obstacle);
}

export function updateObstacleOnBoard(obstacle) {
    const cell = document.getElementById(`cell-${obstacle.x}-${obstacle.y}`);
    if (cell) {
        cell.classList.add('obstacle');
    }
}

export function updateItemOnBoard(item) {
    const cell = document.getElementById(`cell-${item.position.x}-${item.position.y}`);
    if (cell) {
        cell.classList.add('item', `item-${item.item.type}`);
    }
}

export function addItem(item) {
    shared_state.items.push(item);
    updateItemOnBoard(item);
}