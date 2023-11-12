# Scorecard-bot
A whatsapp bot to keep score during group games. built with
whatsapp multidevice api and go.


## Working Idea
- admin sends `chloe \start`, it outputs "starting new games night session."
- on the backend, it creates a new game row. the game row has an id of the new game and when it started
- admin sends `chloe \add 5 to 2348xxx`. it creates a new row on the "leaderboards" table
- with an id, the game_id, phone number of the user and their points
- it will check if an entry exists for that game id and phone number. if it does it will just increment its value
- admin sends `chloe \leaderboard`. It will send the leaderboard for that game session


## Tables
- games: id, timestamps
- leaderboard: id, game_id, phone, score, timestamps