# jianbuquan

����ṹ���ܣ�
1. Ŀ¼�ṹ -- page[���htmlҳ��ģ��] pkg[�������͵������⾲̬�⣬����ʱ����] public[���web��̬����] src[���Դ����]
2. ����ṹ -- main.goΪ����ļ��� dataobj����������ģ�� webhandler������ҳ���߼� weblog��Ϊ��־ģ�� weixin��ʵ���˺�΢�Žӿ�Э��Խ� ���⻹Ӧ���˵�������Դ��redigo���ں�redisͨ��
3. ���нṹ -- go����ʵ��ȫ��ҳ���΢�ŶԽ��߼�������ʹ��redis�洢������ģ�Ͷ���μ�redis���.xlsx
               �����������£�
               ���ں�<---->web������<---->redis
               
������룺
1. ����׼���� ��������linux����windowsϵͳ����Ҫ��װgo���Ի�����git����
2. clone��Ŀ git clone https://github.com/renyangang/jianbuquan.git
3. ����redigo��Ŀ������ʹ��go get���أ�Ҳ����ֱ������zip������ѹ��srcĿ¼�¡�
4. ����GOPATH��������Ϊ src��һ��Ŀ¼�� ��src��ִ��go build ����

�������У�
1. ��װredis��������Ҫ����� 127.0.0.1:8432
2. ����õķ����������� public page ƽ����Ŀ¼���м��ɡ� ����web.log��־�ļ���