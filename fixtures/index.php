<?php

header('x-test: testing');

echo <<<EOT
<!doctype html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Caddy Cache</title>
</head>
<body>
    <a href="/setcookie.php">set cookie</a>
    <a href="/delcookie.php">delete cookie</a>
</body>
</html>
EOT;
