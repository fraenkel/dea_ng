package utils_test

import (
	. "dea/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Utils", func() {
	Describe("RepeatFixed", func() {

		It("repeats every fixed interval", func() {
			ch := make(chan time.Time)
			times := make([]time.Time, 0, 2)

			start := time.Now()
			ticker := RepeatFixed(100*time.Millisecond, func() {
				ch <- time.Now()
			})

			for {
				select {
				case t := <-ch:
					times = append(times, t)
				}

				if len(times) == cap(times) {
					break
				}
			}

			elapsed := time.Now().Sub(start)
			ticker.Stop()

			Expect(elapsed).To(BeNumerically("<", time.Duration(len(times)+1)*100*time.Millisecond))
		})

		It("delay is a fixed rate", func() {
			ch := make(chan time.Time)
			times := make([]time.Time, 0, 3)

			ticker := RepeatFixed(100*time.Millisecond, func() {
				ch <- time.Now()
				time.Sleep(75 * time.Millisecond)
			})

			for {
				select {
				case t := <-ch:
					times = append(times, t)
				}

				if len(times) == cap(times) {
					break
				}
			}

			ticker.Stop()

			for i := 0; i < len(times)-1; i++ {
				gap := times[i+1].Sub(times[i])
				Expect(gap).To(BeNumerically(">=", 95*time.Millisecond))
				Expect(gap).To(BeNumerically("<", 125*time.Millisecond))
			}
		})

		It("stops when the ticker is stopped", func() {
			ch := make(chan time.Time)
			times := make([]time.Time, 0, 3)

			ticker := RepeatFixed(100*time.Millisecond, func() {
				t := time.Now()
				times = append(times, t)
				ch <- t
			})

			select {
			case <-ch:
				ticker.Stop()
			}
			close(ch)

			time.Sleep(200 * time.Millisecond)
			Expect(len(times)).To(Equal(1))
		})
	})

	Describe("Repeat", func() {

		It("repeats every delay interval", func() {
			ch := make(chan time.Time)
			times := make([]time.Time, 0, 2)

			start := time.Now()
			ticker := Repeat(100*time.Millisecond, func() {
				ch <- time.Now()
			})

			for {
				select {
				case t := <-ch:
					times = append(times, t)
				}

				if len(times) == cap(times) {
					break
				}
			}

			elapsed := time.Now().Sub(start)
			ticker.Stop()

			Expect(elapsed).To(BeNumerically("<", time.Duration(len(times)+1)*100*time.Millisecond))
		})

		It("delay is a fixed delay", func() {
			ch := make(chan time.Time)
			times := make([]time.Time, 0, 3)

			ticker := Repeat(100*time.Millisecond, func() {
				ch <- time.Now()
				time.Sleep(75 * time.Millisecond)
			})

			for {
				select {
				case t := <-ch:
					times = append(times, t)
				}

				if len(times) == cap(times) {
					break
				}
			}

			ticker.Stop()

			for i := 0; i < len(times)-1; i++ {
				gap := times[i+1].Sub(times[i])
				Expect(gap).To(BeNumerically(">=", 150*time.Millisecond))
				Expect(gap).To(BeNumerically("<", 190*time.Millisecond))
			}
		})

		It("stops when the ticker is stopped", func() {
			ch := make(chan time.Time)
			times := make([]time.Time, 0, 3)

			ticker := Repeat(100*time.Millisecond, func() {
				t := time.Now()
				times = append(times, t)
				ch <- t
			})

			select {
			case <-ch:
				ticker.Stop()
			}
			close(ch)

			time.Sleep(200 * time.Millisecond)
			Expect(len(times)).To(Equal(1))
		})
	})

	Describe("Timeout", func() {
		It("executes the function", func() {
			err := Timeout(500*time.Millisecond, func() error {
				return nil
			})

			Expect(err).To(BeNil())
		})

		It("reports a timeout if the function takes too long", func() {
			err := Timeout(10*time.Millisecond, func() error {
				time.Sleep(100 * time.Millisecond)
				return nil
			})

			Expect(err).NotTo(BeNil())
		})
	})

	Describe("UUIDString", func() {
		It("creates a UUID without the dashes", func() {
			uuid := UUID()
			Expect(uuid).NotTo(Equal(""))
			Expect(uuid).NotTo(ContainSubstring("-"))
		})
	})
})
