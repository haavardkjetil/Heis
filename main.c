//
//  main.c
//  test
//
//  Created by Kjetil Børs-Lind on 12.01.15.
//  Copyright (c) 2015 Kjetil Børs-Lind. All rights reserved.
//

// gcc 4.7.2 +
// gcc -std=gnu99 -Wall -g -o helloworld_c helloworld_c.c -lpthread

#include <pthread.h>
#include <stdio.h>





int j = 0;

void* func1(){
    for(int i = 0; i < 1000000; i++){
        j++;
    }
    return NULL;
}

void* func2(){
    for(int i = 0; i < 1000000; i++){
        j--;
    }
    return NULL;
}



int main(){
    pthread_t thread1, thread2;
    pthread_create(&thread1, NULL, func1, NULL);
    pthread_create(&thread2, NULL, func2, NULL);
    pthread_join(thread1, NULL);
    pthread_join(thread2, NULL);
    printf("%d", j);
}

/* Go-kode:
 
 package main
 
 import (
 "fmt"
 "time"
 )
 
 
 var j int = 0
 func function1() {
 for i := 0; i < 1000000; i++ {
 time.Sleep(1*time.Nanosecond)
 j++
 }
 }
 
 func function2() {
 for i := 0; i < 1000000; i++ {
 time.Sleep(1*time.Nanosecond)
 j--
 }
 }
 
 
 func main() {
 go function1()
 go function2()
 time.Sleep(1000*time.Millisecond)
 fmt.Println(j)
 }
 
*/