import {shared_state} from "./game_shared.js";
import {sendItemActionRequest, sendMoveRequest, updatePlayerInList} from "./game_action.js";

let serverTimeDiff = 0;

export function handleRoundState(roundState) {
    const state = roundState.state;
    const currentTime = new Date(roundState.currentTime);
    const endTime = new Date(roundState.endTime);

    console.log(`Round state changed to: ${state}`);

    if (state === 'playing') {
        shared_state.playerScores = {};
        Object.keys(shared_state.players).forEach(playerId => {
            shared_state.playerScores[playerId] = 0;
        });
        shared_state.playerScores[shared_state.playerId] = 0;
        updateAllPlayerScores();
        removeWaitingOverlay();
        resumeGame();

    } else if (state === 'waiting' || state === 'preparing' || state === 'ended') {
        pauseGame();
        showWaitingOverlay(`${state.charAt(0).toUpperCase() + state.slice(1)} for next round.`, endTime.getTime() / 1000);
    }

    if (state === 'cleanup') {
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
    // keep scores for show the top player in waiting overlay
    // shared_state.playerScores = {};

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
    const serverEndTime = countdownUpdate.serverEndTime;

    const timerDisplay = document.getElementById('time-left');

    let displayText = '';
    switch (currentState) {
        case 'waiting':
            displayText = 'Waiting for next round';
            break;
        case 'cleanup':
            displayText = `Cleanup time: ${remainingTime}`;
            break;
        case 'preparing':
            displayText = `Preparing time: ${remainingTime}`;
            break;
        case 'playing':
            displayText = `Game remaining time: ${remainingTime}`;
            break;
        case 'ended':
            displayText = 'Game Ended';
            break;
        default:
            displayText = `${currentState} time: ${remainingTime}`;
    }

    timerDisplay.textContent = displayText;

    // Update waiting overlay countdown
    updateWaitingCountdown(serverEndTime);

    // Update top player info in overlay if it exists
    const topPlayerInfo = document.getElementById('top-player-info');
    if (topPlayerInfo) {
        updateTopPlayerInfo(topPlayerInfo);
    }
}

export function showWaitingOverlay(message, nextRoundStart) {
    removeWaitingOverlay();

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

        const topPlayerElement = document.createElement('p');
        topPlayerElement.id = 'top-player-info';
        updateTopPlayerInfo(topPlayerElement);

        const messageElement = document.createElement('p');
        messageElement.textContent = message;

        const countdownElement = document.createElement('p');
        countdownElement.id = 'waiting-countdown';

        contentContainer.appendChild(topPlayerElement);
        contentContainer.appendChild(messageElement);
        contentContainer.appendChild(countdownElement);
        overlay.appendChild(contentContainer);
        document.body.appendChild(overlay);

        if (nextRoundStart) {
            updateWaitingCountdown(nextRoundStart);
        }
    });
}

function updateTopPlayerInfo(element) {
    const topPlayer = shared_state.getTopPlayer();
    if (topPlayer) {
        element.textContent = `Top player: ${topPlayer[0]} (Score: ${topPlayer[1]})`;
    } else {
        element.textContent = '';
    }
}

export function removeWaitingOverlay() {
    const overlay = document.getElementById('waiting-overlay');
    if (overlay) {
        overlay.remove();
    }
    resumeGame();
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

export function handleWaitingNotification(content){
    pauseGame();
    showWaitingOverlay(`${content.message}`, content.nextRoundStart);
}

function getAdjustedServerTime() {
    return Date.now() + serverTimeDiff;
}

export function updateWaitingCountdown(serverEndTime) {
    const countdownElement = document.getElementById('waiting-countdown');
    if (!countdownElement) return;

    const updateCountdown = () => {
        const now = getAdjustedServerTime() / 1000;
        const timeLeft = Math.max(0, Math.floor(serverEndTime - now));

        if (timeLeft > 0) {
            countdownElement.textContent = `Next round starts in ${timeLeft} seconds`;
            requestAnimationFrame(updateCountdown);
        } else {
            countdownElement.textContent = 'Starting soon...';
        }
    };
    updateCountdown();
}