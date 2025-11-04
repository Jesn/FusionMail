import requests


def get_access_token(refresh_token: str, client_id: str) -> str:
    res = requests.post(
        "https://login.microsoftonline.com/common/oauth2/v2.0/token",
        data={
            "client_id": client_id,
            "grant_type": "refresh_token",
            "refresh_token": refresh_token,
            "scope": "https://graph.microsoft.com/.default"
        }
    )
    return res.json()["access_token"]


def print_inbox(access_token: str) -> None:
    res = requests.get(
        "https://graph.microsoft.com/v1.0/me/mailFolders/inbox/messages",
        headers={"Authorization": f"Bearer {access_token}"},
    )
    res.raise_for_status()
    print(res.text)
    for m in res.json().get("value", []):
        print(f"Subject: {m.get('subject')}")
        print(f"From: {m.get('from', {}).get('emailAddress', {}).get('address')}")
        print(f"Text: {m.get('bodyPreview')}")
        # print(f"Html: {m.get('body', {}).get('content')}")
        print(f'\n{"-" * 50}\n', end='')

account = "cohuuexdw097@outlook.com----fqfvqLGz1kIQ----M.C534_BAY.0.U.-CrSmXoA*9zP*UGc7J23aQhYranb0hAF!wbo9ss6P4SN28hlLn3YUwF7s!OrEv2O759zN0zOcrPC8v8erMAshg553ITekSoEIZHIaEiIgjhQ4JIJKdSmfBHSBgmPyv*8o6nMrkgQfzOoMqlY9xlmCDZmfiNebOQgwwCYXBEpi7hEqK*99wZTC32yNOnoEb2hMvvjDePSEio9fbMnaZuzoL6LVka*gz4w5hMR5b058uXtMWGfMsAutjj9mpTuBOc8e7LQ26yLcs*ZLf1XYicLc5V2MPzmv9bL67Mwl3Z7bp7e*6XSrKoiSNCQ0T1p5pz*x9dPDUFl3H0*T!siWR8L*L4QQW61h3kyn6Ngz*zJT*r3fqAvvoAyrJQxWdJ2Kfb4h1lyikdBHQE8Fls9gSqACcfM$----8b4ba9dd-3ea5-4e5f-86f1-ddba2230dcf2"
refresh_token =account.split("----")[2]
client_id = account.split("----")[3]
access_token = get_access_token(refresh_token, client_id)
print_inbox(access_token)