module.exports = {
  describe: function () {
    return { name: "tic-tac-toe", version: "1.0.0" };
  },

  init: function () {
    return {
      board: ["", "", "", "", "", "", "", "", ""],
      turn: "X",
      status: "playing"
    };
  },

  view: function (state) {
    var b = state.board;
    var rows = [
      (b[0] || "\u00B7") + " | " + (b[1] || "\u00B7") + " | " + (b[2] || "\u00B7"),
      (b[3] || "\u00B7") + " | " + (b[4] || "\u00B7") + " | " + (b[5] || "\u00B7"),
      (b[6] || "\u00B7") + " | " + (b[7] || "\u00B7") + " | " + (b[8] || "\u00B7")
    ];
    var boardStr = rows.join("\n---------\n");

    if (state.status === "win") {
      return {
        widgetType: "confirm",
        stepId: "gameover",
        input: {
          title: state.winner + " wins!",
          message: boardStr + "\n\nGood game! Play again?",
          approveText: "Rematch",
          rejectText: "Quit"
        }
      };
    }

    if (state.status === "draw") {
      return {
        widgetType: "confirm",
        stepId: "gameover",
        input: {
          title: "It's a draw!",
          message: boardStr + "\n\nPlay again?",
          approveText: "Rematch",
          rejectText: "Quit"
        }
      };
    }

    var labels = ["Top-Left", "Top-Center", "Top-Right",
                  "Mid-Left", "Center",     "Mid-Right",
                  "Bot-Left", "Bot-Center", "Bot-Right"];
    var options = [];
    for (var i = 0; i < 9; i++) {
      if (!b[i]) options.push(labels[i]);
    }

    return {
      widgetType: "select",
      stepId: "move",
      input: {
        title: "Your turn (X)",
        options: options,
        multi: false,
        searchable: false
      }
    };
  },

  update: function (state, event) {
    if (event.stepId === "gameover") {
      if (event.data && event.data.approved) {
        return {
          board: ["", "", "", "", "", "", "", "", ""],
          turn: "X",
          status: "playing"
        };
      }
      return { done: true, result: { message: "Thanks for playing!" } };
    }

    var labels = ["Top-Left", "Top-Center", "Top-Right",
                  "Mid-Left", "Center",     "Mid-Right",
                  "Bot-Left", "Bot-Center", "Bot-Right"];
    var pick = event.data && event.data.selectedSingle;
    var idx = -1;
    for (var i = 0; i < labels.length; i++) {
      if (labels[i] === pick) { idx = i; break; }
    }
    if (idx < 0 || state.board[idx]) return state;

    state.board[idx] = "X";

    if (checkWin(state.board, "X")) {
      state.status = "win";
      state.winner = "X";
      return state;
    }
    if (isFull(state.board)) {
      state.status = "draw";
      return state;
    }

    // Computer's turn (O) — simple strategy
    var move = computerMove(state.board);
    if (move >= 0) {
      state.board[move] = "O";
      if (checkWin(state.board, "O")) {
        state.status = "win";
        state.winner = "O";
        return state;
      }
      if (isFull(state.board)) {
        state.status = "draw";
        return state;
      }
    }

    return state;
  }
};

function checkWin(b, p) {
  var lines = [
    [0,1,2],[3,4,5],[6,7,8],
    [0,3,6],[1,4,7],[2,5,8],
    [0,4,8],[2,4,6]
  ];
  for (var i = 0; i < lines.length; i++) {
    if (b[lines[i][0]] === p && b[lines[i][1]] === p && b[lines[i][2]] === p) return true;
  }
  return false;
}

function isFull(b) {
  for (var i = 0; i < 9; i++) { if (!b[i]) return false; }
  return true;
}

function computerMove(b) {
  var lines = [
    [0,1,2],[3,4,5],[6,7,8],
    [0,3,6],[1,4,7],[2,5,8],
    [0,4,8],[2,4,6]
  ];
  // 1. Win if possible
  for (var i = 0; i < lines.length; i++) {
    var m = tryLine(b, lines[i], "O");
    if (m >= 0) return m;
  }
  // 2. Block X from winning
  for (var i = 0; i < lines.length; i++) {
    var m = tryLine(b, lines[i], "X");
    if (m >= 0) return m;
  }
  // 3. Take center
  if (!b[4]) return 4;
  // 4. Take a corner
  var corners = [0, 2, 6, 8];
  for (var i = 0; i < corners.length; i++) {
    if (!b[corners[i]]) return corners[i];
  }
  // 5. Take any open cell
  for (var i = 0; i < 9; i++) {
    if (!b[i]) return i;
  }
  return -1;
}

function tryLine(b, line, player) {
  var count = 0;
  var empty = -1;
  for (var i = 0; i < 3; i++) {
    if (b[line[i]] === player) count++;
    else if (!b[line[i]]) empty = line[i];
  }
  if (count === 2 && empty >= 0) return empty;
  return -1;
}
