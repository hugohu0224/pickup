import {shared_state} from "./game_shared.js";
import {sendItemActionRequest, sendMoveRequest, updatePlayerInList} from "./game_action.js";

let serverTimeDiff = 0;

export function handleRoundState(roundState) {
    const state = roundState.state;
    const currentTime = new Date(roundState.currentTime);
    const endTime = new Date(roundState.endTime);

    console.log(`Round state changed to: ${state}`);

    removeWaitingOverlay();

    if (state === 'playing') {
        shared_state.playerScores = {};
        Object.keys(shared_state.players).forEach(playerId => {
            shared_state.playerScores[playerId] = 0;
        });
        shared_state.playerScores[shared_state.playerId] = 0;
        updateAllPlayerScores();
        removeWaitingOverlay();
        resumeGame();
    } else if (state === 'waiting') {
        pauseGame();
        showWaitingOverlay(`Waiting for the next round. \nProcessing: ${state}`, true);
    } else if (state === 'preparing' || state === 'ended') {
        pauseGame();
        showWaitingOverlay(`Waiting for the next round. \nProcessing: ${state}`);
    }

    if (state === 'cleanup') {
        removeWaitingOverlay();
        showWaitingOverlay(`Waiting for the next round. \nProcessing: ${state}`);
        resetGameData();
    }

    updateCountdown({
        remainingTime: Math.floor((endTime - currentTime) / 1000),
        currentState: state,
        serverEndTime: endTime.getTime() / 1000
    });
}

export function updateAllPlayerScores() {
    Object.keys(shared_state.players).forEach(playerId => {
        updatePlayerInList(playerId);
    });
}

export function resetGameData() {
    shared_state.playerPosition = {x: 0, y: 0};
    shared_state.lastConfirmedPosition = {x: 0, y: 0};
    shared_state.players = {};
    shared_state.obstacles = [];
    shared_state.items = [];

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

    console.log('Game data reset');
}

export function updateCountdown(countdownUpdate) {
    const remainingTime = countdownUpdate.remainingTime;
    const currentState = countdownUpdate.currentState;

    const timerDisplay = document.getElementById('time-left');

    let displayText = '';
    switch (currentState) {
        case 'waiting':
            displayText = `Showing Top Score: ${remainingTime}s`;
            break;
        case 'cleanup':
            displayText = `Cleanup time: ${remainingTime}s`;
            break;
        case 'preparing':
            displayText = `Preparing time: ${remainingTime}s`;
            break;
        case 'playing':
            displayText = `Remaining time: ${remainingTime}s`;
            break;
        case 'ended':
            displayText = 'Game Ended';
            break;
        default:
            displayText = `${currentState} time: ${remainingTime}s`;
    }

    timerDisplay.textContent = displayText;
}

export function showWaitingOverlay(message, showTopPlayer = false) {
    removeWaitingOverlay();

    if (document.getElementById('waiting-overlay')) {
        console.warn('Waiting overlay already exists, not creating a new one');
        return;
    }

    requestAnimationFrame(() => {
        const overlay = document.createElement('div');
        overlay.id = 'waiting-overlay';
        overlay.style.cssText = `
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background-color: rgba(128, 128, 128, 0.7);
            display: flex;
            justify-content: center;
            align-items: center;
            z-index: 1000;
            color: white;
            font-size: 24px;
            text-align: center;
        `;

        const contentContainer = document.createElement('div');
        contentContainer.style.cssText = `
            display: flex;
            flex-direction: column;
            align-items: center;
            gap: 20px;
        `;

        const messageElement = document.createElement('p');
        messageElement.textContent = message;

        contentContainer.appendChild(messageElement);

        if (showTopPlayer) {
            const topPlayerElement = document.createElement('p');
            updateTopPlayerInfo(topPlayerElement);
            contentContainer.appendChild(topPlayerElement);
        }

        overlay.appendChild(contentContainer);
        document.body.appendChild(overlay);
    });
}

function updateTopPlayerInfo(element) {
    const topPlayer = shared_state.getTopPlayer();
    if (topPlayer) {
        element.innerHTML = `<span class="top-player">Top Player: ${topPlayer[0]} (Score: ${topPlayer[1]})</span>`;
    } else {
        element.textContent = '';
    }
    element.style.fontSize = '24px';
}

export function removeWaitingOverlay() {
    const overlays = document.querySelectorAll('#waiting-overlay');
    overlays.forEach(overlay => overlay.remove());
}

export function pauseGame() {
    document.removeEventListener('keydown', handleKeyPress);
}

export function resumeGame() {
    document.addEventListener('keydown', handleKeyPress);
}

const directionMap = {
    'ArrowUp': 'up',
    'ArrowDown': 'down',
    'ArrowLeft': 'left',
    'ArrowRight': 'right'
};

export function handleKeyPress(event) {
    const direction = directionMap[event.key];
    if (direction) {
        sendMoveRequest(direction);
    } else if (event.key === ' ') {
        sendItemActionRequest();
    }
}

export function handleWaitingNotification(content) {
    pauseGame();
    removeWaitingOverlay();
    showWaitingOverlay(`${content.message}`, content.nextRoundStart);
}
