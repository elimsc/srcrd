因为log共享一个`buf[]`, 因此读写的时候要**加锁**

log的打印函数使用`fmt.Sprintxx`来生成string, 这个操作是非常费时的