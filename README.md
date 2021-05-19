This is a quick thing I threw together to update WoW addons from Curse. I find other "addon updater" clients are way too clunky.

To use, modify the addons.json file included in this repo and then run the program in the working directory where the addons.json file is located.

The addon names in the addons.json file are taken from the curseforge URL for the addon and the project ID. If the project ID is 1 and the slug in the URL is "my-addon" when viewing the addon on curseforge, the name in addons.json should be "1-my-addon".

This program saves which file version it downloaded for each addon into an "addons.lock.json" file, and will use that file to prevent downloading the same version of an addon multiple times.
