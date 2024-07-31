let countdowns = {};
let serverTimeDiff = 0;

function checkRoomStatus(room) {
    fetch(`/v1/game/room-status?roomId=${room}`)
        .then(response => response.json())
        .then(data => {
            console.log(`Received data for room ${room}:`, data);
            const serverTime = data.serverTime ? new Date(data.serverTime).getTime() : Date.now();
            const nextRoundStart = new Date(data.nextRoundStart).getTime();
            const currentTime = Date.now();

            serverTimeDiff = data.serverTime ? currentTime - serverTime : 0;
            console.log(`Current time: ${currentTime}, Server time: ${serverTime}, Difference: ${serverTimeDiff}ms`);

            updateRoomStatus(room, nextRoundStart, data.state);
        })
        .catch(error => {
            console.error('Error:', error);
            updateRoomStatusError(room);
        });
}

function updateRoomStatus(room, nextRoundStart, state) {
    if (countdowns[room]) {
        clearInterval(countdowns[room]);
    }

    const statusElement = document.getElementById(`status-${room}`);
    const countdownElement = document.getElementById(`countdown-${room}`);
    const btnElement = document.getElementById(`btn-${room}`);

    const updateStatus = () => {
        const currentTime = Date.now();
        const now = currentTime - serverTimeDiff;
        const remainingTime = Math.max(0, Math.floor((nextRoundStart - now) / 1000));

        let statusText, countdownText, canJoin;
        let reversePreparingTime = 50
        if (remainingTime > reversePreparingTime) {
            statusText = "Game Preparing";
            countdownText = `${remainingTime-reversePreparingTime}s`;
            canJoin = false;
        } else {
            statusText = "You can join now <br>(but wait for the next round)";
            countdownText = `${remainingTime}s`;
            canJoin = true;
        }

        statusElement.innerHTML = statusText;
        countdownElement.textContent = countdownText;
        btnElement.disabled = !canJoin;
        btnElement.textContent = canJoin ? "Join Room" : "Waiting";

        if (now >= nextRoundStart) {
            checkRoomStatus(room);
        }
    };

    updateStatus();
    countdowns[room] = setInterval(updateStatus, 1000);
}

function updateRoomStatusError(room) {
    const statusElement = document.getElementById(`status-${room}`);
    const countdownElement = document.getElementById(`countdown-${room}`);
    const btnElement = document.getElementById(`btn-${room}`);

    statusElement.textContent = "Error";
    countdownElement.textContent = "Unable to fetch status";
    btnElement.disabled = true;
    btnElement.textContent = "Unavailable";
}

function formatTime(seconds) {
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = seconds % 60;
    return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
}

function initializeRooms() {
    ['A', 'B'].forEach(room => {
        const btnElement = document.getElementById(`btn-${room}`);
        if (!btnElement) {
            console.error(`Button element for room ${room} not found`);
            return;
        }
        btnElement.addEventListener('click', () => {
            if (!btnElement.disabled) {
                window.location.href = `/v1/game/page?roomId=${room}`;
            }
        });
        checkRoomStatus(room);
    });
}

window.addEventListener('load', initializeRooms);

window.onerror = function(message, source, lineno, colno, error) {
    console.error("Global error:", message, "at", source, ":", lineno, ":", colno);
    console.error("Error object:", error);
};