package gigachatllm

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"backend/mail/internal/client/imap"
	"backend/mail/internal/config"
	"backend/mail/internal/storage"
)

func TestGenerate(t *testing.T) {
	config, err := config.New("../../../../.env")
	require.NoError(t, err, "config new")

	llm := New(config.LLM)

	question := "Что там за проблема с оплатой у Ивана и почему у них стоит работа?"
	resp, err := llm.Generate(context.Background(), question, generateTestData())
	require.NoError(t, err, "generate llm response")

	t.Log("Result:\n", resp)
}

func generateTestData() []storage.ScoredPoint {
	now := time.Now()

	return []storage.ScoredPoint{
		{
			Score: 0.95, // Самый важный целевой запрос
			Payload: &imap.Letter{
				Envelope: imap.Envelope{
					UID:     10425,
					Date:    now.Add(-10 * time.Minute), // 10 минут назад
					Subject: "Ошибка при оплате тарифа Бизнес",
					From: imap.Address{
						Name:    "Иван Иванов",
						Mailbox: "ivan.ivanov",
						Host:    "company.ru",
					},
				},
				Body: "Здравствуйте! Наш бухгалтер пытается оплатить счет за ваш сервис, но банк выдает ошибку 403. Доступ заблокирован, работа стоит. Пожалуйста, пришлите новую ссылку на оплату или проверьте статус.",
			},
		},
		{
			Score: 0.82, // Важное клиентское письмо
			Payload: &imap.Letter{
				Envelope: imap.Envelope{
					UID:     10426,
					Date:    now.Add(-1 * time.Hour), // 1 час назад
					Subject: "Вопрос по API интеграции",
					From: imap.Address{
						Name:    "Технический Лид",
						Mailbox: "tech-lead",
						Host:    "startup.io",
					},
				},
				Body: "Приветствую. Пытаемся настроить вебхуки через ваше API, но не находим в документации, какой формат JSON вы присылаете при изменении статуса. Можете скинуть актуальный пример структуры запроса?",
			},
		},
		{
			Score: 0.68, // Спорное письмо (возможно коммерческое предложение)
			Payload: &imap.Letter{
				Envelope: imap.Envelope{
					UID:     10427,
					Date:    now.Add(-4 * time.Hour),
					Subject: "Сотрудничество и интеграция платежей",
					From: imap.Address{
						Name:    "Елена Маркетолог",
						Mailbox: "e.smirnova",
						Host:    "pay-gate.net",
					},
				},
				Body: "Добрый день! Наша компания занимается интернет-эквайрингом. Мы заметили, что у вашего сервиса бывают сбои при оплате. Предлагаем рассмотреть интеграцию нашего платежного шлюза с комиссией от 1%.",
			},
		},
		{
			Score: 0.45, // Низкий приоритет (рассылка, на которую ИИ отвечать не должен)
			Payload: &imap.Letter{
				Envelope: imap.Envelope{
					UID:     10428,
					Date:    now.Add(-12 * time.Hour),
					Subject: "Дайджест обновлений платформы за май",
					From: imap.Address{
						Name:    "Команда Облака",
						Mailbox: "news",
						Host:    "cloud-service.com",
					},
				},
				Body: "Привет! Мы обновили панель управления и ускорили работу баз данных в 2 раза. Подробнее обо всех изменениях читайте в нашем блоге по ссылке ниже.",
			},
		},
		{
			Score: 0.12, // Очевидный спам, который зацепился поиском случайно
			Payload: &imap.Letter{
				Envelope: imap.Envelope{
					UID:     10429,
					Date:    now.Add(-24 * time.Hour),
					Subject: "Заработок без вложений для ИТ-специалистов",
					From: imap.Address{
						Name:    "Алексей",
						Mailbox: "super-job",
						Host:    "spam-marketing.biz",
					},
				},
				Body: "Предлагаем уникальную схему пассивного дохода! Вывод денег на любую карту каждый день. Никаких рисков, начните зарабатывать от 5000 рублей прямо сейчас.",
			},
		},
	}
}
