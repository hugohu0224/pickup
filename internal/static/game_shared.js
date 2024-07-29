export const shared_state = {
    gridSize: 15,
    socket: null,
    playerPosition: {x: 0, y: 0},
    lastConfirmedPosition: {x: 0, y: 0},
    playerId: null,
    players: {},
    playerScores: {},
    obstacles: [],
    items: [],
    isGameInitialized: false,

    // DOM
    gameBoard: null,
    playerList: null,
    timerDisplay: null,

    // func
    getTopPlayer: function() {
        const entries = Object.entries(this.playerScores);
        if (entries.length === 0) return null;
        return entries.reduce((top, current) => current[1] > top[1] ? current : top);
    }
}