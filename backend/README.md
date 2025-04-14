

- Whenever an user connects to the websocket
    - username
- call the init func
    - init the client (board state)
    - start the reading channel for reading in msgs
        - whenever another user makes a change 
        the reading channel will read that in and then
        apply it
        - for ex, write -> {pos : (3, 5), rbg : (0,0,0)} 
        apply that change to the board


- run game func 
    - take in (uuid, pos, rgb)

    - for all client != uuid 
        - send them the pos and rgb