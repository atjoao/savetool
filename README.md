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
    savetool (-game "<nome do jogo>") -service github -github <token>+<user/repo>+(branch) -saves "<pasta de save do jogo>" -kp=<true|false> -- %COMMAND%
    ```

- () > opcional
- -kp default = `true`

- Este tem de ficar em ordem de acordo com o exemplo para prevenir erros
- game flag é opcional SE EXISITR SteamAppId ou SteamGameId em variaveis do processo (por norma existe se for iniciado pela Steam)

# Args

```
-saves string
    Caminho para a pasta de saves
-service string
    Nome do serviço (github / catbox)
-kp boolean
    Manter os saves guardados no diretório do jogo - game_dir/saves/<timestamp>.zip (predefinido: true)
-catbox string
    Configuração do Catbox || obrigatório dependendo do serviço escolhido
-github string
    Configuração do GitHub: token+user/repo+branch (branch opcional) || obrigatório dependendo do serviço escolhido
-game string
    Identificador do nome do jogo (obrigatório para github)
```

# Github Tutorial

Cria um repo no github como `saves`\
Depois clica [aqui](https://github.com/settings/personal-access-tokens)\
Cria uma token, nas definições seleciona.\
`Expiration date` > seleciona `No Expiration`\
`Only select repositories` > seleciona o teu repo `saves`\
`Permissions` > seleciona `Contents` > Escolhe `Read and Write`\
No final terás uma token que podes usar para `-github token+user/saves`

# CatBox Tutorial

Criar uma conta\
Ir a [`Manage Account`](https://catbox.moe/user/manage.php)\
Copiar `User Hash`\
Ir a [`Manage Albuns`](https://catbox.moe/user/manage_albums.php)\
[`Criar um album`](https://catbox.moe/user/view.php)\
Clicar em `Add to album`\
Clicar no dropdown e selecionar `Create new album`\
Dar um nome aleatorio (de preferencia nome do jogo para indentificação facil)\
Copiar resultado do album (texto verde) que vai ser `https://catbox.moe/c/xxxxxx`\
Copiar a parte do `xxxxxx`\
No final terás `-catbox userhash+xxxxxx`

# Notas sobre link2ea:// protocol

Este protocolo é para ser usado com [link2ea](https://github.com/atjoao/Link2EA)\
Eu não sei como funciona o oficial, nem testei com o oficial.\
Por isso não é meu problema (este é apenas para windows, se estiver certo, no linux vai iniciar debaixo do wine/proton por isso prosseguirá normalmente)