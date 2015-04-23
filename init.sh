echo "Starter sublime..."
echo
cd
cd /var/run/user/10264/gvfs/smb-share:server=sambaad.stud.ntnu.no,share=hhholta/Sublime
xterm -e ./sublime_text / &

echo "setting path GUI..."
echo
cd
cd Heis

xterm -e sh initLogger.sh &

chmod +777 run.sh

