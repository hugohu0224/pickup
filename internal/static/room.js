let countdowns = {};

function checkRoomStatus(room) {
    fetch(`/v1/game/room-status?roomId=${room}`)
        .then(response => response.json())
        .then(data => {
            startCountdown(room, data.nextRoundStart);
        })
        .catch(error => {
            console.error('Error:', error);
            alert("Error checking room status, please try again later.");
        });
}

function startCountdown(room, nextRoundStart) {
    if (countdowns[room]) {
        clearInterval(countdowns[room]);
    }

    const countdownElement = document.getElementById(`countdown-${room}`);
    const updateCountdown = () => {
        const now = new Date().getTime();
        const currentSecond = Math.floor((now / 1000) % 60);

        if (currentSecond >= 5 && currentSecond <= 60) {
            countdownElement.innerHTML = "Allow join now.";
            return;
        }

        const distance = nextRoundStart - now;
        if (distance <= 0) {
            checkRoomStatus(room);
            return;
        }

        const seconds = Math.floor(distance / 1000) % 60;
        countdownElement.innerHTML = `Preparing: ${seconds} seconds`;
    };

    updateCountdown();
    countdowns[room] = setInterval(updateCountdown, 1000);
}

function initializeRooms() {
    ['A', 'B'].forEach(room => {
        const roomElement = document.getElementById(`room-${room}`);
        roomElement.addEventListener('click', () => {
            const countdownElement = document.getElementById(`countdown-${room}`);
            if (countdownElement.innerHTML === "Allow join now.") {
                window.location.href = `/v1/game/page?roomId=${room}`;
            } else {
                alert("Game is preparing, please wait.");
            }
        });
        checkRoomStatus(room);
        setInterval(() => checkRoomStatus(room), 15000); // 每15秒检查一次状态
    });
}

window.addEventListener('load', initializeRooms);