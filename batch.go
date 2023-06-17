package migrator

func Batch(total int, batchSize int, fn func(skip int, limit int) error) error {
	totalBatch := total / batchSize
	if total%batchSize != 0 {
		totalBatch += 1
	}
	for i := 0; i < totalBatch; i++ {
		err := fn(i*batchSize, batchSize)
		if err != nil {
			return err
		}
	}
	return nil
}
