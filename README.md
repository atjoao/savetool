# SaveTool
Uma ferramenta para saves em qualquer plataforma

# Executar
- Abrir propriedades de jogo
- Definir opções de lançamento

Exemplo catbox
    ```
    savetool -service catbox -catbox <userhash>+<albumid> -saves "<pasta de save do jogo>" -kp=<true|false> -- %COMMAND%
    ```

Exemplo github
    ```
    savetool -game <nome do jogo> -service github -github <token>+<repo>+<branch> -saves "<pasta de save do jogo>" -kp=<true|false> -- %COMMAND%
    ```
- Este tem de ficar em ordem de acordo com o exemplo para prevenir erros

